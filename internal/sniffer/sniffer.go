package sniffer

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"go-kafka-sol-listener/internal/wallet"
)

// Sniffer is responsible for processing messages and determining whether they should be sent to a webhook.
type Sniffer struct {
	// walletManager manages the list of wallets to check against.
	walletManager *wallet.WalletManager
	// mutex ensures thread-safe operations when sending matched messages.
	mutex sync.Mutex
	// webhookURL is the endpoint where matched messages are sent.
	webhookURL string
}

// NewSniffer initializes a new Sniffer with a wallet manager and a webhook URL.
func NewSniffer(walletManager *wallet.WalletManager, webhookURL string) *Sniffer {
	return &Sniffer{
		walletManager: walletManager,
		webhookURL:    webhookURL,
	}
}

// HandleMessages processes a batch of messages to find matches and send them to the webhook.
func (s *Sniffer) HandleMessages(messages []map[string]interface{}) {
	// Sort the messages by Block.Timestamp (lowest to highest) as the absolute first step.
	sort.Slice(messages, func(i, j int) bool {
		blockI, okI := messages[i]["Block"].(map[string]interface{})
		blockJ, okJ := messages[j]["Block"].(map[string]interface{})
		if okI && okJ {
			timestampI, okTI := blockI["Timestamp"].(float64)
			timestampJ, okTJ := blockJ["Timestamp"].(float64)
			if okTI && okTJ {
				return timestampI < timestampJ
			}
		}
		return false // Default order if Block.Timestamp is invalid
	})

	// Record the start timestamp.
	timestampStart := time.Now().Unix()

	// matchedMessages stores messages that match the wallet list.
	matchedMessages := []map[string]interface{}{}

	// Prepare a list to store timestamps from the Block field.
	var blockTimestamps []int64

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

		// Check if the signer exists in the wallet list.
		if s.walletManager.WalletExists(signer) {
			matchedMessages = append(matchedMessages, message)
			log.Println("Match found for signer!")
		}

		// Extract the Block.Timestamp field if it exists.
		block, ok := message["Block"].(map[string]interface{})
		if ok {
			if blockTimestamp, ok := block["Timestamp"].(float64); ok {
				blockTimestamps = append(blockTimestamps, int64(blockTimestamp))
			}
		}
	}

	// Determine the key timestamps from the batch.
	var timestamp1, timestampMiddle, timestampLast int64
	if len(blockTimestamps) > 0 {
		timestamp1 = blockTimestamps[0]
		timestampLast = blockTimestamps[len(blockTimestamps)-1]
		timestampMiddle = blockTimestamps[len(blockTimestamps)/2]
	}

	// Record the end timestamp.
	timestampEnd := time.Now().Unix()

	// Calculate and log metrics.
	s.logMetrics(timestampStart, timestampEnd, timestamp1, timestampMiddle, timestampLast)

	// Send matched messages to the webhook.
	s.sendMatchedMessages(matchedMessages)
}

// sendMatchedMessages sends matched messages to the webhook URL.
func (s *Sniffer) sendMatchedMessages(messages []map[string]interface{}) {
	// Lock ensures no other goroutine modifies the messages during this operation.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, message := range messages {
		// Convert the message to JSON format.
		jsonData, err := json.Marshal(message)
		if err != nil {
			log.Printf("Failed to marshal message: %v\n", err)
			continue
		}

		// Send the JSON data to the webhook.
		resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("Failed to send message to webhook: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		// Log if the webhook does not return a successful status.
		if resp.StatusCode != http.StatusOK {
			log.Printf("Webhook returned non-OK status: %s\n", resp.Status)
		}
	}
}

// logMetrics calculates and logs the latency metrics.
func (s *Sniffer) logMetrics(timestampStart, timestampEnd, timestamp1, timestampMiddle, timestampLast int64) {
	log.Printf("Sniffer Latency: %d seconds\n", timestampEnd-timestampStart)
	if timestamp1 > 0 {
		log.Printf("Batch Latency: %d seconds\n", timestampLast-timestamp1)
		log.Printf("ConsumerLatency1: %d seconds\n", timestampStart-timestamp1)
		log.Printf("ConsumerLatency2: %d seconds\n", timestampStart-timestampMiddle)
		log.Printf("ConsumerLatency3: %d seconds\n", timestampStart-timestampLast)
		log.Printf("Kafka Server Latency: %d seconds\n", (timestampEnd-timestamp1)-(timestampLast-timestamp1))
		log.Printf("Total Latency: %d seconds\n", timestampEnd-timestamp1)
	} else {
		log.Println("No valid timestamps found in the batch for latency calculations.")
	}
}
