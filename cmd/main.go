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
	walletManager := wallet.NewWalletManager(cfg.Application.WalletListURL)

	// Start WalletManager update loop
	go walletManager.UpdateWallets()

	// Periodically print the wallet list
	go func() {
		for {
			wallets := walletManager.GetWalletList()
			log.Printf("Current wallet list: %v\n", wallets)
			time.Sleep(time.Duration(cfg.Application.WalletUpdateIntervalMs) * time.Millisecond)
		}
	}()

	// Initialize Sniffer with webhook URL
	snifferInstance := sniffer.NewSniffer(walletManager, cfg.Application.WebhookURL)

	// Restart logic for the Kafka consumer
	for {
		fmt.Println("Starting Kafka consumer...")
		err := consumer.StartConsumer(cfg, snifferInstance)
		if err != nil {
			log.Printf("Kafka consumer encountered an error: %v. Restarting in 5 seconds...", err)
			time.Sleep(5 * time.Second)
		}
	}
}
