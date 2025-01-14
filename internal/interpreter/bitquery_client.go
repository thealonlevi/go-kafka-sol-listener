package interpreter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// FetchTokenSupply queries BitQuery for token information and returns the token name and supply.
func FetchTokenSupply(mintAddress string) (string, string, error) {
	if bitqueryToken == "" {
		return "", "", fmt.Errorf("BitQuery token is not initialized")
	}

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
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bitqueryToken))

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
