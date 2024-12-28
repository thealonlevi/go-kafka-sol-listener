package sniffer

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"

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
	// matchedMessages stores messages that match the wallet list.
	matchedMessages := []map[string]interface{}{}

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
	}

	// Append the last message of the batch to matchedMessages, even if it doesn't match.
	if len(messages) > 0 {
		lastMessage := messages[len(messages)-1]
		matchedMessages = append(matchedMessages, lastMessage)
		log.Println("Added last message of the batch to matched messages.")
	}

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
