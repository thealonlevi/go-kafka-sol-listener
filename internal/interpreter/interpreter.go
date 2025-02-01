package interpreter

import (
	"encoding/json"
	"fmt"
	"go-kafka-sol-listener/internal/config"
	"go-kafka-sol-listener/internal/utils"
	"log"
	"net/http"
)

var pythonInterpreter string
var swapDetectorScript string
var bitqueryToken string

// Initialize the BitQuery token and script paths from the configuration.
func InitializeInterpreterConfig(cfg *config.Config) {
	pythonInterpreter = cfg.Interpreter.Python
	swapDetectorScript = cfg.Interpreter.SwapDetectorScript
	bitqueryToken = cfg.Interpreter.BitqueryToken
	log.Printf("Python interpreter set to: %s", pythonInterpreter)
	log.Printf("Swap detector script set to: %s", swapDetectorScript)
	log.Printf("BitQuery token initialized.")
}

// ProcessMessage handles the input message, enriches it with BitQuery data, and sends the results to the webhook.
func ProcessMessage(jsonData []byte, webhookURL string, transferWebhookURL string) error {
	log.Println("Starting ProcessMessage")

	// Parse the message to extract the transaction signature.
	var message map[string]interface{}
	if err := json.Unmarshal(jsonData, &message); err != nil {
		return fmt.Errorf("failed to parse JSON data: %w", err)
	}

	signature, ok := extractSignature(message)
	if !ok {
		return fmt.Errorf("transaction signature not found")
	}

	// Check if the transaction has already been processed.
	if !utils.IsUnprocessed(signature) {
		log.Printf("Skipping already processed signature: %s", signature)
		return nil
	}

	// Mark the signature as being processed.
	utils.AddSignature(signature)

	// Invoke the Python script for swap detection.
	result, err := invokePythonScript(jsonData)
	if err != nil {
		return fmt.Errorf("failed to invoke Python script: %w", err)
	}

	log.Printf("Python script output: %s", result)

	var swapDetails map[string]interface{}
	if err := json.Unmarshal([]byte(result), &swapDetails); err != nil {
		return fmt.Errorf("failed to parse Python script output: %w", err)
	}

	// Check if a swap was detected.
	swapDetected, ok := swapDetails["swapDetected"].(bool)
	if !ok || !swapDetected {
		// COMMENT says: "In this scenario, it should refer it to transferWebhookURL"
		log.Println("No swap detected. Sending to transfer webhook...")

		resp, err := sendToWebhook(swapDetails, transferWebhookURL)
		if err != nil {
			return fmt.Errorf("failed to send details to transfer webhook: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("transfer webhook returned non-OK status: %s", resp.Status)
		}
		return nil
	}

	log.Printf("Swap detected: %v", swapDetails)

	// Send enriched details to the main webhook.
	log.Printf("Sending enriched details to webhook: %s", webhookURL)
	resp, err := sendToWebhook(swapDetails, webhookURL)
	if err != nil {
		return fmt.Errorf("failed to send enriched details to webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned non-OK status: %s", resp.Status)
	}

	return nil
}
