package sniffer

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"go-kafka-sol-listener/internal/wallet"
)

type Sniffer struct {
	walletManager *wallet.WalletManager
	mutex         sync.Mutex
	webhookURL    string
}

func NewSniffer(walletManager *wallet.WalletManager, webhookURL string) *Sniffer {
	return &Sniffer{
		walletManager: walletManager,
		webhookURL:    webhookURL,
	}
}

func (s *Sniffer) HandleMessages(messages []map[string]interface{}) {
	matchedMessages := []map[string]interface{}{}

	for _, message := range messages {
		transaction, ok := message["Transaction"].(map[string]interface{})
		if !ok {
			log.Println("Transaction field missing or invalid")
			continue
		}

		signer, ok := transaction["Signer"].(string)
		if !ok {
			log.Println("Signer field missing or invalid")
			continue
		}

		if s.walletManager.WalletExists(signer) {
			matchedMessages = append(matchedMessages, message)
			log.Println("Match found for signer!")
		}
	}

	s.sendMatchedMessages(matchedMessages)
}

func (s *Sniffer) sendMatchedMessages(messages []map[string]interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, message := range messages {
		jsonData, err := json.Marshal(message)
		if err != nil {
			log.Printf("Failed to marshal message: %v\n", err)
			continue
		}

		resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("Failed to send message to webhook: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Webhook returned non-OK status: %s\n", resp.Status)
		}
	}
}
