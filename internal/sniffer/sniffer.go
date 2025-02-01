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
	walletManager      *wallet.WalletManager   // Manages the list of wallets to check against.
	mutex              sync.Mutex              // Ensures thread-safe operations.
	webhookURL         string                  // The endpoint where matched messages are sent.
	metricsHandler     *metrics.MetricsHandler // Handles metrics aggregation and reporting.
	saveMatches        string                  // Controls whether to save matched messages.
	transferWebhookURL string                  // The endpoint where transfer-related messages are sent.
}

// NewSniffer initializes a new Sniffer with a wallet manager, webhook URL, metrics handler, and saveMatches flag.
func NewSniffer(walletManager *wallet.WalletManager, webhookURL string, metricsHandler *metrics.MetricsHandler, saveMatches string, transferWebhookURL string) *Sniffer {
	return &Sniffer{
		walletManager:      walletManager,
		webhookURL:         webhookURL,
		metricsHandler:     metricsHandler,
		saveMatches:        saveMatches,
		transferWebhookURL: transferWebhookURL,
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
		// 1. Check if Transaction.Status.Success is true.
		success, ok := getTransactionSuccess(message)
		if !ok || !success {
			continue
		}

		// 2. Extract the transaction field from the message.
		transaction, ok := message["Transaction"].(map[string]interface{})
		if !ok {
			continue
		}

		// 3. Extract the signer field from the transaction.
		signer, ok := transaction["Signer"].(string)
		if !ok {
			continue
		}

		// 4. Extract the signature field from the transaction.
		signature, ok := transaction["Signature"].(string)
		if !ok {
			continue
		}

		// 5. Lock mutex to ensure thread safety when accessing shared resources.
		s.mutex.Lock()

		// 6. Check if the signer exists in the wallet list.
		if s.walletManager.WalletExists(signer) {
			// Signer exists: Process as a transfer.
			log.Println("Match found for signer and unprocessed signature! Forwarding to interpreter.")
			utils.AddSignature(signature) // Mark the signature as being processed.

			// Save the match if saveMatches is enabled.
			if s.saveMatches != "off" {
				s.saveMatchToFile(signature, message)
			}

			// Unlock mutex before starting goroutine.
			s.mutex.Unlock()

			// Forward to interpreter.
			go s.processWithInterpreter(message)
		} else {
			// Signer does not exist: Check BalanceUpdates for Token.Owner matches.

			// Extract BalanceUpdates
			balanceUpdates, ok := message["BalanceUpdates"].([]map[string]interface{})
			if !ok {
				s.mutex.Unlock()
				continue
			}

			// Ensure there are fewer than 20 BalanceUpdates to prevent excessive processing.
			if len(balanceUpdates) >= 20 {
				s.mutex.Unlock()
				continue
			}

			// Iterate over BalanceUpdates to check Token.Owner
			matched := false
			for _, balanceUpdate := range balanceUpdates {
				balanceUpdateData, ok := balanceUpdate["BalanceUpdate"].(map[string]interface{})
				if !ok {
					continue
				}
				accountData, ok := balanceUpdateData["Account"].(map[string]interface{})
				if !ok {
					continue
				}
				tokenData, ok := accountData["Token"].(map[string]interface{})
				if !ok || tokenData == nil {
					continue
				}
				owner, ok := tokenData["Owner"].(string)
				if !ok {
					continue
				}

				// Check if Token.Owner exists in the wallet manager.
				if s.walletManager.WalletExists(owner) {
					log.Println("Match found for Token.Owner and unprocessed signature! Forwarding to interpreter.")
					utils.AddSignature(signature) // Mark the signature as being processed.

					// Save the match if saveMatches is enabled.
					if s.saveMatches != "off" {
						s.saveMatchToFile(signature, message)
					}

					// Unlock mutex before starting goroutine.
					s.mutex.Unlock()

					// Forward to interpreter.
					go s.processWithInterpreter(message)
					matched = true
					break // Exit loop after first match.
				}
			}

			if !matched {
				// Unlock mutex as we didn't process
				s.mutex.Unlock()
			}
		}

		// 7. Record metrics regardless of processing.
		s.recordMetrics(message)
	}
}

// getTransactionSuccess extracts the Transaction.Status.Success value from a message.
func getTransactionSuccess(message map[string]interface{}) (bool, bool) {
	transaction, ok := message["Transaction"].(map[string]interface{})
	if !ok {
		return false, false
	}
	status, ok := transaction["Status"].(map[string]interface{})
	if !ok {
		return false, false
	}
	success, ok := status["Success"].(bool)
	return success, ok
}

// recordMetrics extracts timestamps and logs metrics for the message.
func (s *Sniffer) recordMetrics(message map[string]interface{}) {
	blockTimestamp, ok := getBlockTimestamp(message)
	if !ok {
		log.Println("Failed to extract Block.Timestamp for metrics.")
		return
	}

	localTimestamp := time.Now().Unix()                           // Current time in Unix seconds
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

// processWithInterpreter forwards the message to the interpreter for swap or transfer detection.
func (s *Sniffer) processWithInterpreter(message map[string]interface{}) {
	// Convert the message to JSON format.
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message for interpreter: %v\n", err)
		return
	}

	// Call the interpreter function with the JSON message, the main webhook URL, and the transfer webhook URL.
	err = interpreter.ProcessMessage(jsonData, s.webhookURL, s.transferWebhookURL)
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
