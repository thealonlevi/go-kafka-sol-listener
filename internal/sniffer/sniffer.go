package sniffer

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"go-kafka-sol-listener/internal/wallet"
)

type Sniffer struct {
	walletManager *wallet.WalletManager
	mutex         sync.Mutex
	outputFile    string
}

func NewSniffer(walletManager *wallet.WalletManager, outputFile string) *Sniffer {
	return &Sniffer{
		walletManager: walletManager,
		outputFile:    outputFile,
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
			log.Println("BINGOO!!!!!")
		}
	}

	s.saveMatchedMessages(matchedMessages)
}

func (s *Sniffer) saveMatchedMessages(messages []map[string]interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	file, err := os.OpenFile(s.outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open file: %v\n", err)
		return
	}
	defer file.Close()

	for _, message := range messages {
		jsonData, err := json.Marshal(message)
		if err != nil {
			log.Printf("Failed to marshal message: %v\n", err)
			continue
		}

		if _, err := file.Write(append(jsonData, '\n')); err != nil {
			log.Printf("Failed to write message to file: %v\n", err)
		}
	}
}
