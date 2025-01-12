package interpreter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-kafka-sol-listener/internal/utils"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

// Reference to the SOL-to-USD cache
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

// getInstanceUID fetches the current instance UID from the utils package.
func getInstanceUID() string {
	uid := utils.GetInstanceUID()
	log.Println("interpreter.go: Instance UID: ", uid)
	return uid
}

// FetchTokenSupply queries BitQuery for token information and returns the token name and supply.
func FetchTokenSupply(mintAddress string) (string, string, error) {
	url := "https://streaming.bitquery.io/eap"

	query := fmt.Sprintf(`{
		Solana {
			TokenSupplyUpdates(
				limit: { count: 1 },
				orderBy: { descending: Block_Time },
				where: { TokenSupplyUpdate: { Currency: { MintAddress: { is: "%s" } } } }
			) {
				TokenSupplyUpdate {
					Amount
					Currency {
						MintAddress
						Name
					}
					PreBalance
					PostBalance
				}
			}
		}
	}`, mintAddress)

	payload := strings.NewReader(fmt.Sprintf(`{"query": %q, "variables": "{}"}`, query))

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer ory_at_hAVdOlEkgl523t31iZllc5JtzmvWssIxakbplbh7AK4.dxrnRFAaMr9Jinapcr-cK-p7JIehLdgDfuPxQVm6uPc")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	log.Println(string(body))
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Validate and traverse the JSON response
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("missing 'data' field")
	}

	solana, ok := data["Solana"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("missing 'Solana' field")
	}

	tokenSupplyUpdates, ok := solana["TokenSupplyUpdates"].([]interface{})
	if !ok || len(tokenSupplyUpdates) == 0 {
		return "", "", fmt.Errorf("no 'TokenSupplyUpdates' found")
	}

	tokenSupplyUpdate, ok := tokenSupplyUpdates[0].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("missing 'TokenSupplyUpdate' in response")
	}

	updateData, ok := tokenSupplyUpdate["TokenSupplyUpdate"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid 'TokenSupplyUpdate' structure")
	}

	currency, ok := updateData["Currency"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("missing 'Currency' field in 'TokenSupplyUpdate'")
	}

	name, ok := currency["Name"].(string)
	if !ok {
		return "", "", fmt.Errorf("'Name' field is missing or invalid in 'Currency'")
	}

	postBalance, ok := updateData["PostBalance"].(string)
	if !ok {
		return "", "", fmt.Errorf("'PostBalance' field is missing or invalid in 'TokenSupplyUpdate'")
	}

	return name, postBalance, nil
}

// ProcessMessage handles the input message, enriches it with BitQuery data, and sends the results to the webhook.
func ProcessMessage(jsonData []byte, webhookURL string) error {
	log.Println("Salam")
	// Parse the message to extract the transaction signature.
	var message map[string]interface{}
	if err := json.Unmarshal(jsonData, &message); err != nil {
		return fmt.Errorf("failed to parse JSON data: %w", err)
	}
	log.Println("Hello")
	signature, ok := extractSignature(message)
	if !ok {
		return fmt.Errorf("transaction signature not found")
	}

	// Check if the transaction has already been processed.
	if utils.IsUnprocessed(signature) {
		log.Printf("Skipping already processed signature: %s", signature)
		return nil
	}
	log.Println("One")
	// Mark the signature as being processed.
	utils.AddSignature(signature)

	// Invoke the Python script for swap detection.
	result, err := invokePythonScript(jsonData)
	if err != nil {
		log.Println("WTF")
		return fmt.Errorf("failed to invoke Python script: %w", err)
	}
	log.Println("Two")
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

	// Fetch token supply and update symbol.
	token2, _ := details["Token2"].(map[string]interface{})
	if token2 != nil {
		mintAddress := token2["Mint"].(string)
		log.Println("uno")
		name, supply, err := FetchTokenSupply(mintAddress)
		log.Println("duo")
		if err != nil {
			log.Printf("Failed to fetch token supply: %v", err)
		} else {
			token2["Symbol"] = name
			token2["TokenSupply"] = supply
		}
	} else {
		log.Println("Alekum")
	}

	// Add instanceUID to the details.
	details["instanceUID"] = getInstanceUID()

	// Log the enriched details for debugging.
	logEnrichedDetails(details)

	// Send enriched details to the webhook.
	log.Printf("Sending enriched details to webhook: %s\n", webhookURL)
	resp, err := sendToWebhook(details, webhookURL)
	if err != nil {
		return fmt.Errorf("failed to send enriched details to webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned non-OK status: %s", resp.Status)
	}

	return nil
}

// extractSignature extracts the transaction signature from the message.
func extractSignature(message map[string]interface{}) (string, bool) {
	transaction, ok := message["Transaction"].(map[string]interface{})
	if !ok {
		return "", false
	}

	signature, ok := transaction["Signature"].(string)
	return signature, ok
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

// sendToWebhook sends the JSON data **double-encoded** to the webhook URL,
// so that the receiving Lambda gets an event like { "body": "<your JSON>" }.
func sendToWebhook(details map[string]interface{}, webhookURL string) (*http.Response, error) {
	// 1) Marshal your enriched details into a JSON string.
	innerJSON, err := json.Marshal(details)
	if err != nil {
		log.Printf("Failed to marshal enriched details: %v\n", err)
		return nil, fmt.Errorf("failed to marshal enriched details: %w", err)
	}

	// 2) Build a top-level object with a "body" field containing that JSON string.
	topLevel := map[string]string{
		"body": string(innerJSON),
	}

	// 3) Marshal the top-level object.
	finalPayload, err := json.Marshal(topLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to double-encode payload: %w", err)
	}

	// 4) POST the double-encoded JSON to the webhook.
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(finalPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to send enriched details to webhook: %w", err)
	}

	return resp, nil
}

// invokePythonScript executes the Python script and returns the result of the swap detection.
func invokePythonScript(jsonData []byte) (string, error) {
	cmd := exec.Command("python3", "scripts/swapdetector.py")
	cmd.Stdin = bytes.NewBuffer(jsonData)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute Python script: %v\nOutput: %s", err, output.String())
	}
	return output.String(), nil
}
