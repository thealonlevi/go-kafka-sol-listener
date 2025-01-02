package main

import (
	"fmt"
	"go-kafka-sol-listener/internal/config"
	"go-kafka-sol-listener/internal/consumer"
	"go-kafka-sol-listener/internal/sniffer"
	"go-kafka-sol-listener/internal/wallet"
	"log"
	"time"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	fmt.Printf("Loaded configuration: %+v\n", cfg)

	// Initialize WalletManager
	walletManager := wallet.NewWalletManager("https://s3hq4ph0s0.execute-api.eu-west-1.amazonaws.com/getWalletList")

	// Start WalletManager update loop
	go walletManager.UpdateWallets()

	// Periodically print the wallet list
	go func() {
		for {
			wallets := walletManager.GetWalletList()
			log.Printf("Current wallet list: %v\n", wallets)
			time.Sleep(1 * time.Minute)
		}
	}()

	// Initialize Sniffer with webhook URL
	snifferInstance := sniffer.NewSniffer(walletManager, "https://s3hq4ph0s0.execute-api.eu-west-1.amazonaws.com/transactionWebhook")

	// Restart logic for the Kafka consumer
	for {
		fmt.Println("Starting Kafka consumer...")
		err := consumer.StartConsumer(cfg, snifferInstance) // Updated to return an error
		if err != nil {
			log.Printf("Kafka consumer encountered an error: %v. Restarting in 5 seconds...", err)
			time.Sleep(5 * time.Second) // Wait before restarting
		}
	}
}
