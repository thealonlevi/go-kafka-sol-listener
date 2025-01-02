package interpreter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

// ProcessMessage handles the input message, invokes the Python script for swap detection, and sends the results to the webhook.
func ProcessMessage(jsonData []byte, webhookURL string) error {
	// Invoke the Python script for swap detection.
	result, err := invokePythonScript(jsonData)
	if err != nil {
		return fmt.Errorf("failed to invoke Python script: %w", err)
	}

	log.Printf("Python script output: %s", result)

	if result == "No swap detected" {
		log.Println("No swap detected.")
		return nil
	}

	log.Printf("Swap detected: %s\n", result)

	// Prepare the payload for the webhook.
	payload := map[string]interface{}{
		"SwapDetails":     result,
		"OriginalMessage": json.RawMessage(jsonData),
	}

	wrappedPayload := map[string]interface{}{
		"body": payload,
	}

	jsonPayload, err := json.Marshal(wrappedPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Send the detected swap to the webhook.
	log.Println(webhookURL)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send to webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned non-OK status: %s", resp.Status)
	}

	return nil
}

// invokePythonScript executes the Python script and returns the result of the swap detection.
func invokePythonScript(jsonData []byte) (string, error) {
	// Prepare the command to execute the Python script.
	cmd := exec.Command("python3", "scripts/swapdetector.py")

	// Provide the JSON data as input to the script.
	cmd.Stdin = bytes.NewBuffer(jsonData)

	// Capture the script's output.
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	// Run the command.
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute Python script: %v\nOutput: %s", err, output.String())
	}

	// Return the script's output as a string.
	return output.String(), nil
}
