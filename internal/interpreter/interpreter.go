package interpreter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sync"
)

// Reference to the SOL-to-USD cache from main.go
var solToUsdCache struct {
	rate  float64
	mutex sync.RWMutex
}

// SetSolToUsdCache allows updating the SOL-to-USD rate in the global cache.
func SetSolToUsdCache(rate float64) {
	solToUsdCache.mutex.Lock()
	defer solToUsdCache.mutex.Unlock()
	solToUsdCache.rate = rate
}

// getCachedSolToUsdRate retrieves the current SOL-to-USD rate from the cache.
func getCachedSolToUsdRate() (float64, error) {
	solToUsdCache.mutex.RLock()
	defer solToUsdCache.mutex.RUnlock()
	if solToUsdCache.rate == 0 {
		return 0, fmt.Errorf("SOL-to-USD rate is not available")
	}
	return solToUsdCache.rate, nil
}

// ProcessMessage handles the input message, invokes the Python script for swap detection, and sends the results to the webhook.
func ProcessMessage(jsonData []byte, webhookURL string) error {
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
		log.Println("No swap detected.")
		return nil
	}

	log.Printf("Swap detected: %v\n", swapDetails)

	// Enrich the swap details with USD calculations.
	details, ok := swapDetails["details"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid swap details structure")
	}
	enrichSwapDetails(details)

	// Log the enriched details for debugging.
	logEnrichedDetails(details)

	// Marshal enriched details as the body of the request.
	jsonPayload, err := json.MarshalIndent(map[string]interface{}{"body": details}, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal enriched details: %v\n", err)
		return fmt.Errorf("failed to marshal enriched details: %w", err)
	}

	// Send the enriched details to the webhook.
	log.Printf("Sending enriched details to webhook: %s\n", webhookURL)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send enriched details to webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned non-OK status: %s", resp.Status)
	}

	return nil
}

// enrichSwapDetails calculates missing USD values and enriches the swap details.
func enrichSwapDetails(details map[string]interface{}) {
	rate, err := getCachedSolToUsdRate()
	if err != nil {
		log.Printf("Failed to fetch SOL-to-USD rate: %v\n", err)
		return
	}

	// Token1 enrichment
	if token1, ok := details["Token1"].(map[string]interface{}); ok {
		if amount, ok := token1["Amount"].(float64); ok {
			token1["AmountUSD"] = amount * rate
		}
	}

	// Token2 enrichment
	if token2, ok := details["Token2"].(map[string]interface{}); ok {
		if amount, ok := token2["Amount"].(float64); ok && amount > 0 {
			if token1, ok := details["Token1"].(map[string]interface{}); ok {
				if token1Amount, ok := token1["Amount"].(float64); ok && token1Amount > 0 {
					// Calculate SOL/TOKEN ratio
					solPerToken := token1Amount / amount

					// Calculate USD/TOKEN ratio
					usdPerToken := solPerToken * rate

					// Enrich Token2 fields
					token2["AmountSOL"] = solPerToken * amount
					token2["AmountUSD"] = usdPerToken * amount
				}
			}
		}

		if postBalance, ok := token2["PostSwapBalance"].(float64); ok && postBalance > 0 {
			if token1, ok := details["Token1"].(map[string]interface{}); ok {
				if token1Amount, ok := token1["Amount"].(float64); ok && token1Amount > 0 {
					solPerToken := token1Amount / token2["Amount"].(float64)
					usdPerToken := solPerToken * rate
					token2["PostSwapBalanceUSD"] = postBalance * usdPerToken
				}
			}
		}
	}

	// Fee enrichment
	if fee, ok := details["Fee"].(map[string]interface{}); ok {
		if amount, ok := fee["Amount"].(float64); ok {
			fee["AmountUSD"] = amount * rate
		}
	}
}

// logEnrichedDetails neatly prints the enriched swap details in JSON format.
func logEnrichedDetails(details map[string]interface{}) {
	enrichedJSON, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		log.Printf("Failed to format enriched details: %v\n", err)
		return
	}
	log.Printf("Enriched Swap Details:\n%s\n", string(enrichedJSON))
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
