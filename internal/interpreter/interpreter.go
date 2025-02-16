package interpreter

import (
	"encoding/json"
	"fmt"
	"go-kafka-sol-listener/internal/config"
	"go-kafka-sol-listener/internal/utils"
	"io/ioutil"
	"log"
	"net/http"
)

var pythonInterpreter string
var swapDetectorScript string
var bitqueryToken string

// InitializeInterpreterConfig initializes the BitQuery token and script paths from the configuration.
func InitializeInterpreterConfig(cfg *config.Config) {
	pythonInterpreter = cfg.Interpreter.Python
	swapDetectorScript = cfg.Interpreter.SwapDetectorScript
	bitqueryToken = cfg.Interpreter.BitqueryToken
	log.Printf("Python interpreter set to: %s", pythonInterpreter)
	log.Printf("Swap detector script set to: %s", swapDetectorScript)
	log.Printf("BitQuery token initialized.")
}

// ProcessMessage handles incoming messages, invokes the Python script, and routes swaps/transfers.
func ProcessMessage(jsonData []byte, webhookURL string, transferWebhookURL string, databaseEndpoint string) error {
	log.Println("Starting ProcessMessage")

	// Parse the input message.
	var message map[string]interface{}
	if err := json.Unmarshal(jsonData, &message); err != nil {
		return fmt.Errorf("failed to parse JSON data: %w", err)
	}

	// Extract the transaction signature.
	signature, ok := extractSignature(message)
	if !ok {
		return fmt.Errorf("transaction signature not found")
	}

	// Check if this transaction has already been processed.
	if !utils.IsUnprocessed(signature) {
		log.Printf("Skipping already processed signature: %s", signature)
		return nil
	}

	// Mark the signature as processed.
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

	// Determine if a swap was detected.
	swapDetected, ok := swapDetails["swapDetected"].(bool)
	if !ok {
		return fmt.Errorf("failed to determine swapDetected status")
	}

	// Prepare the packaged data which will be sent to the sol-transaction API.
	var packagedData = map[string]interface{}{
		"data": swapDetails,
	}

	if !swapDetected {
		// Mark the operation type as "transfer" when no swap is detected.
		packagedData["type"] = "transfer"

		// Send the packaged data to the sol-transaction API.
		log.Printf("Sending packaged data to sol-transaction API: %s", databaseEndpoint)
		resp2, err2 := sendToWebhook(packagedData, databaseEndpoint)
		if err2 != nil {
			return fmt.Errorf("failed to send packaged data to sol-transaction API: %w", err2)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			return fmt.Errorf("sol-transaction API returned non-OK status: %s", resp2.Status)
		}

		log.Println("No swap detected. Sending to transfer webhook...")
		resp, err := sendToWebhook(swapDetails, transferWebhookURL)
		if err != nil {
			return fmt.Errorf("failed to send details to transfer webhook: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("transfer webhook returned non-OK status: %s", resp.Status)
		}

	} else {
		// Mark the operation type as "swap".
		packagedData["type"] = "swap"

		// Send the packaged data to the sol-transaction API.
		log.Printf("Sending packaged data to sol-transaction API: %s", databaseEndpoint)
		resp2, err2 := sendToWebhook(packagedData, databaseEndpoint)
		if err2 != nil {
			return fmt.Errorf("failed to send packaged data to sol-transaction API: %w", err2)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			return fmt.Errorf("sol-transaction API returned non-OK status: %s", resp2.Status)
		}

		// Attempt to read the realized_pnl from sol-transaction API's response.
		bodyBytes, err := ioutil.ReadAll(resp2.Body)
		if err != nil {
			return fmt.Errorf("failed to read sol-transaction API response body: %w", err)
		}

		// Example top-level structure: {"status": "received", "realized_pnl": 123.45}
		var solTxResp map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &solTxResp); err != nil {
			return fmt.Errorf("failed to unmarshal sol-transaction API response: %w", err)
		}

		// If realized_pnl is present, attach it to swapDetails before sending it forward
		if rPnl, hasRPnl := solTxResp["realized_pnl"]; hasRPnl {
			swapDetails["realized_pnl"] = rPnl
		}

		// Swap detected: send to the main webhook.
		log.Printf("Swap detected: %v", swapDetails)
		log.Printf("Sending enriched details to webhook: %s", webhookURL)

		resp, err := sendToWebhook(swapDetails, webhookURL)
		if err != nil {
			return fmt.Errorf("failed to send enriched details to webhook: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("webhook returned non-OK status: %s", resp.Status)
		}
	}

	return nil
}
