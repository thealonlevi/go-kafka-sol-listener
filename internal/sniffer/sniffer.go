package sniffer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go-kafka-sol-listener/internal/interpreter"
	"go-kafka-sol-listener/internal/metrics"
	"go-kafka-sol-listener/internal/utils"
	"go-kafka-sol-listener/internal/wallet"
)

// Sniffer is responsible for processing messages and determining whether they should be sent to a webhook.
type Sniffer struct {
	walletManager  *wallet.WalletManager   // Manages the list of wallets to check against.
	mutex          sync.Mutex              // Ensures thread-safe operations.
	webhookURL     string                  // The endpoint where matched messages are sent.
	metricsHandler *metrics.MetricsHandler // Handles metrics aggregation and reporting.
}

// NewSniffer initializes a new Sniffer with a wallet manager, webhook URL, and metrics handler.
func NewSniffer(walletManager *wallet.WalletManager, webhookURL string, metricsHandler *metrics.MetricsHandler) *Sniffer {
	return &Sniffer{
		walletManager:  walletManager,
		webhookURL:     webhookURL,
		metricsHandler: metricsHandler,
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
			log.Println("Match found for signer and unprocessed signature! Forwarding to interpreter.")
			utils.AddSignature(signature) // Mark the signature as being processed.

			// Save the entire JSON message to a file.
			s.saveMatchToFile(signature, message)

			// Forward to interpreter.
			go s.processWithInterpreter(message)
		}

		s.recordMetrics(message)
	}
}

// recordMetrics extracts timestamps and logs metrics for the message.
func (s *Sniffer) recordMetrics(message map[string]interface{}) {
	blockTimestamp, ok := getBlockTimestamp(message)
	if !ok {
		log.Println("Failed to extract Block.Timestamp for metrics.")
		return
	}

	localTimestamp := time.Now().Unix()
	go s.metricsHandler.AddMetric(blockTimestamp, localTimestamp) // Non-blocking metrics recording.
}

// saveMatchToFile saves the matched JSON message to a file in the "matches" folder.
func (s *Sniffer) saveMatchToFile(signature string, message map[string]interface{}) {
	// Create the "matches" folder if it doesn't exist.
	folderPath := "matches"
	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		log.Printf("Failed to create matches folder: %v\n", err)
		return
	}

	// Convert the message to JSON format.
	jsonData, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal message to JSON: %v\n", err)
		return
	}

	// Construct the file path using the signature.
	filePath := filepath.Join(folderPath, fmt.Sprintf("%s.json", signature))

	// Write the JSON data to the file.
	if err := os.WriteFile(filePath, jsonData, os.ModePerm); err != nil {
		log.Printf("Failed to write match to file: %v\n", err)
		return
	}

	log.Printf("Match saved to file: %s\n", filePath)
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
