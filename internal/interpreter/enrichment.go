// enrichment.go

package interpreter

import (
	"log"
)

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
func enrichSwapDetails(details map[string]interface{}, rate float64) {
	// If rate is invalid (e.g., zero), skip enrichment.
	if rate <= 0 {
		log.Println("enrichSwapDetails: Invalid SOL-to-USD rate, skipping enrichment.")
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

	// Fetch token supply for Token2
	if mintAddress, ok := token2["Mint"].(string); ok {
		name, supply, err := FetchTokenSupply(mintAddress)
		if err != nil {
			log.Printf("Failed to fetch token supply for %s: %v", mintAddress, err)
		} else {
			token2["Symbol"] = name
			token2["TokenSupply"] = supply
		}
	}
}
