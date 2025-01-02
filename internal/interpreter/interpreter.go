package interpreter

import (
	"encoding/json"
	"fmt"
	"math"
)

type Account struct {
	Address    string `json:"Address"`
	IsSigner   bool   `json:"IsSigner"`
	IsWritable bool   `json:"IsWritable"`
	Token      *Token `json:"Token"`
}

type Token struct {
	Decimals int    `json:"Decimals"`
	Mint     string `json:"Mint"`
	Owner    string `json:"Owner"`
}

type BalanceUpdate struct {
	Account     Account `json:"Account"`
	PostBalance int64   `json:"PostBalance"`
	PreBalance  int64   `json:"PreBalance"`
}

type Currency struct {
	Decimals    int    `json:"Decimals"`
	MintAddress string `json:"MintAddress"`
	Name        string `json:"Name"`
}

type Update struct {
	BalanceUpdate BalanceUpdate `json:"BalanceUpdate"`
	Currency      Currency      `json:"Currency"`
}

type Transaction struct {
	Signer string `json:"Signer"`
}

type Input struct {
	BalanceUpdates []Update    `json:"BalanceUpdates"`
	Transaction    Transaction `json:"Transaction"`
}

func DetectSwap(jsonData []byte) (string, error) {
	var input Input
	if err := json.Unmarshal(jsonData, &input); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	signer := input.Transaction.Signer
	filteredUpdates := []Update{}

	for _, update := range input.BalanceUpdates {
		if (update.BalanceUpdate.Account.Address == signer ||
			(update.BalanceUpdate.Account.Token != nil && update.BalanceUpdate.Account.Token.Owner == signer)) &&
			update.BalanceUpdate.PostBalance != update.BalanceUpdate.PreBalance {
			filteredUpdates = append(filteredUpdates, update)
		}
	}

	if len(filteredUpdates) < 2 {
		return "No swap detected", nil
	}

	firstUpdate := filteredUpdates[0]
	lastUpdate := filteredUpdates[len(filteredUpdates)-1]

	firstAmountRaw := firstUpdate.BalanceUpdate.PostBalance - firstUpdate.BalanceUpdate.PreBalance
	lastAmountRaw := lastUpdate.BalanceUpdate.PostBalance - lastUpdate.BalanceUpdate.PreBalance

	firstAmount := float64(firstAmountRaw) / math.Pow10(firstUpdate.Currency.Decimals)
	lastAmount := float64(lastAmountRaw) / math.Pow10(lastUpdate.Currency.Decimals)

	token1Name := firstUpdate.Currency.Name
	if token1Name == "" {
		token1Name = firstUpdate.Currency.MintAddress
	}
	token2Name := lastUpdate.Currency.Name
	if token2Name == "" {
		token2Name = lastUpdate.Currency.MintAddress
	}

	var spent, received string
	if firstAmount < 0 {
		spent = fmt.Sprintf("-%.5f %s", math.Abs(firstAmount), token1Name)
		received = fmt.Sprintf("+%.5f %s", math.Abs(lastAmount), token2Name)
	} else {
		spent = fmt.Sprintf("-%.5f %s", math.Abs(lastAmount), token2Name)
		received = fmt.Sprintf("+%.5f %s", math.Abs(firstAmount), token1Name)
	}

	return fmt.Sprintf("Swapped: %s %s", received, spent), nil
}
