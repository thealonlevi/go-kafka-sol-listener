package sniffer

import (
	"encoding/json"
	"log"
	"sort"
	"sync"

	"go-kafka-sol-listener/internal/interpreter"
	"go-kafka-sol-listener/internal/utils"
	"go-kafka-sol-listener/internal/wallet"
)

// Sniffer is responsible for processing messages and determining whether they should be sent to a webhook.
type Sniffer struct {
	walletManager *wallet.WalletManager // Manages the list of wallets to check against.
	mutex         sync.Mutex            // Ensures thread-safe operations.
	webhookURL    string                // The endpoint where matched messages are sent.
}

// NewSniffer initializes a new Sniffer with a wallet manager and a webhook URL.
func NewSniffer(walletManager *wallet.WalletManager, webhookURL string) *Sniffer {
	return &Sniffer{
		walletManager: walletManager,
		webhookURL:    webhookURL,
	}
}

// HandleMessages processes a batch of messages to find matches and forward them to the interpreter.
func (s *Sniffer) HandleMessages(messages []map[string]interface{}) {
	// Sort the messages by Block.Timestamp (lowest to highest).
	sort.Slice(messages, func(i, j int) bool {
		tsI, okI := getBlockTimestamp(messages[i])
		tsJ, okJ := getBlockTimestamp(messages[j])
		if okI && okJ {
			return tsI < tsJ
		}
		return false
	})

	// Iterate through the messages to process them.
	for _, message := range messages {
		// Extract the transaction field from the message.
		transaction, ok := message["Transaction"].(map[string]interface{})
		if !ok {
			log.Println("Transaction field missing or invalid")
			continue
		}

		// Extract the signer field from the transaction.
		signer, ok := transaction["Signer"].(string)
		if !ok {
			log.Println("Signer field missing or invalid")
			continue
		}

		// Extract the signature field from the transaction.
		signature, ok := transaction["Signature"].(string)
		if !ok {
			log.Println("Signature field missing or invalid")
			continue
		}

		// Check if the signer exists in the wallet list and the signature is unprocessed.
		if s.walletManager.WalletExists(signer) {
			log.Println("Match!")
			if utils.IsUnprocessed(signature) {
				log.Println("Match found for signer and unprocessed signature! Forwarding to interpreter.")
				utils.AddSignature(signature) // Mark the signature as being processed.
				go s.processWithInterpreter(message)
			} else {
				log.Printf("Duplicate signature detected: %s. Skipping.\n", signature)
			}
		}
	}
}

// processWithInterpreter forwards the message to the interpreter for swap detection.
func (s *Sniffer) processWithInterpreter(message map[string]interface{}) {
	// Convert the message to JSON format.
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message for interpreter: %v\n", err)
		return
	}

	// Call the interpreter function with the JSON message and webhook URL.
	err = interpreter.ProcessMessage(jsonData, s.webhookURL)
	if err != nil {
		log.Printf("Interpreter processing failed: %v\n", err)
	}
}

// getBlockTimestamp safely extracts the Block.Timestamp value from a message.
func getBlockTimestamp(message map[string]interface{}) (int64, bool) {
	block, ok := message["Block"].(map[string]interface{})
	if !ok {
		return 0, false
	}
	timestamp, ok := block["Timestamp"].(float64)
	if !ok {
		return 0, false
	}
	return int64(timestamp), true
}
