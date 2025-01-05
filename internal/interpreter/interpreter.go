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

	details = ensureTokenOrder(details)
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

// ensureTokenOrder ensures that Token1 is always SOL and Token2 is the other token.
func ensureTokenOrder(details map[string]interface{}) map[string]interface{} {
	token1, _ := details["Token1"].(map[string]interface{})
	token2, _ := details["Token2"].(map[string]interface{})

	if token2 == nil || token1 == nil {
		return details
	}

	if token1["Symbol"] != "SOL" && token2["Symbol"] == "SOL" {
		// Swap Token1 and Token2 to ensure SOL is Token1
		details["Token1"] = token2
		details["Token2"] = token1
	}

	return details
}

// enrichSwapDetails calculates missing USD values and enriches the swap details.
func enrichSwapDetails(details map[string]interface{}) {
	rate, err := getCachedSolToUsdRate()
	if err != nil {
		log.Printf("Failed to fetch SOL-to-USD rate: %v\n", err)
		return
	}

	token1, _ := details["Token1"].(map[string]interface{})
	token2, _ := details["Token2"].(map[string]interface{})

	if token1 == nil || token2 == nil {
		return
	}

	// Enrich Token1
	if amountChange, ok := token1["AmountChange"].(float64); ok {
		token1["AmountSOL"] = amountChange
		token1["AmountUSD"] = amountChange * rate
	}
	if postBalance, ok := token1["PostSwapBalance"].(float64); ok {
		token1["PostSwapBalanceSOL"] = postBalance
		token1["PostSwapBalanceUSD"] = postBalance * rate
	}
	if preBalance, ok := token1["PreSwapBalance"].(float64); ok {
		token1["PreSwapBalanceSOL"] = preBalance
		token1["PreSwapBalanceUSD"] = preBalance * rate
	}

	// Calculate SOL to Token2 ratio
	ratio := 0.0
	if token1Change, ok := token1["AmountChange"].(float64); ok && token1Change != 0 {
		if token2Change, ok := token2["AmountChange"].(float64); ok {
			ratio = -token1Change / token2Change
		}
	}

	// Enrich Token2
	if amountChange, ok := token2["AmountChange"].(float64); ok {
		token2["AmountSOL"] = amountChange * ratio
		token2["AmountUSD"] = (amountChange * ratio) * rate
	}
	if postBalance, ok := token2["PostSwapBalance"].(float64); ok {
		token2["PostSwapBalanceSOL"] = postBalance * ratio
		token2["PostSwapBalanceUSD"] = (postBalance * ratio) * rate
		if token2["PostSwapBalanceUSD"].(float64) < 0.01 {
			token2["PostSwapBalanceUSD"] = 0.0
			token2["PostSwapBalanceSOL"] = 0.0
		}
	}
	if preBalance, ok := token2["PreSwapBalance"].(float64); ok {
		token2["PreSwapBalanceSOL"] = preBalance * ratio
		token2["PreSwapBalanceUSD"] = (preBalance * ratio) * rate
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
	cmd := exec.Command("python", "scripts/swapdetector.py")
	cmd.Stdin = bytes.NewBuffer(jsonData)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute Python script: %v\nOutput: %s", err, output.String())
	}
	return output.String(), nil
}
