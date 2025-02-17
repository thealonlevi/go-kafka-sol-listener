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
		// Build a debug payload that includes the entire Python script output
		// and the entire input we sent to the Python script.
		debugPayload := map[string]interface{}{
			"source": "interpreter",
			"debug_info": map[string]interface{}{
				"python_script_input":  string(jsonData),
				"python_script_output": result,
			},
		}

		// Send this debug info to the error endpoint.
		_, debugErr := sendToWebhook(debugPayload, "http://13.49.221.13:8082/api/errors")
		if debugErr != nil {
			log.Printf("Failed to send debug info to http://13.49.221.13:8082/api/errors: %v", debugErr)
		}

		return fmt.Errorf("failed to parse Python script output: %w", err)
	}

	// Determine if a swap was detected.
	swapDetected, ok := swapDetails["swapDetected"].(bool)
	if !ok {
		return fmt.Errorf("failed to determine swapDetected status")
	}

	// Prepare the packaged data which will be sent to the sol-transaction API.
	var packagedData map[string]interface{}

	if !swapDetected {
		// No swap detected: send to the transfer webhook.
		packagedData = map[string]interface{}{
			"data": swapDetails,
			"type": "transfer",
		}

		// Send to the sol-transaction API.
		log.Printf("Sending packaged data to sol-transaction API: %s", databaseEndpoint)
		resp2, err2 := sendToWebhook(packagedData, databaseEndpoint)
		if err2 != nil {
			return fmt.Errorf("failed to send packaged data to sol-transaction API: %w", err2)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			return fmt.Errorf("sol-transaction API returned non-OK status: %s", resp2.Status)
		}

		// Now send to the transfer webhook.
		log.Println("No swap detected. Sending to transfer webhook...")
		resp, err := sendToWebhook(swapDetails, transferWebhookURL)
		if err != nil {
			return fmt.Errorf("failed to send details to transfer webhook: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("transfer webhook returned non-OK status: %s", resp.Status)
		}

		// Also send to the dashboard API.
		dashboardPayload := map[string]interface{}{
			"type": "transfer",
			"data": swapDetails,
		}
		dashResp, dashErr := sendToWebhook(dashboardPayload, "http://13.49.221.13:8082/api/data")
		if dashErr != nil {
			return fmt.Errorf("failed to send details to dashboard: %w", dashErr)
		}
		defer dashResp.Body.Close()
		if dashResp.StatusCode != http.StatusOK {
			return fmt.Errorf("dashboard returned non-OK status: %s", dashResp.Status)
		}

	} else {
		// A swap was detected.
		packagedData = map[string]interface{}{
			"data": swapDetails,
			"type": "swap",
		}

		// Send to the sol-transaction API.
		log.Printf("Sending packaged data to sol-transaction API: %s", databaseEndpoint)
		resp2, err2 := sendToWebhook(packagedData, databaseEndpoint)
		if err2 != nil {
			return fmt.Errorf("failed to send packaged data to sol-transaction API: %w", err2)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			return fmt.Errorf("sol-transaction API returned non-OK status: %s", resp2.Status)
		}

		// Read the response from sol-transaction; set realized_pnl.
		var respBody map[string]interface{}
		if err := json.NewDecoder(resp2.Body).Decode(&respBody); err == nil {
			swapDetails["pnl_data"] = respBody["pnl_data"]
		}

		// Send the enriched details to the main webhook.
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

		// Also send to the dashboard API.
		dashboardPayload := map[string]interface{}{
			"type": "swap",
			"data": swapDetails,
		}
		dashResp, dashErr := sendToWebhook(dashboardPayload, "http://13.49.221.13:8082/api/data")
		if dashErr != nil {
			return fmt.Errorf("failed to send details to dashboard: %w", dashErr)
		}
		defer dashResp.Body.Close()
		if dashResp.StatusCode != http.StatusOK {
			return fmt.Errorf("dashboard returned non-OK status: %s", dashResp.Status)
		}
	}

	return nil
}
