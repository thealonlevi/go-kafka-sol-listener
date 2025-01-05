package main

import (
	"encoding/json"
	"fmt"
	"go-kafka-sol-listener/internal/config"
	"go-kafka-sol-listener/internal/consumer"
	"go-kafka-sol-listener/internal/interpreter" // Import interpreter to update cache
	"go-kafka-sol-listener/internal/sniffer"
	"go-kafka-sol-listener/internal/wallet"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// fetchSolToUsdRate fetches the current SOL-to-USD rate and updates the interpreter cache.
func fetchSolToUsdRate() {
	updateRate() // Fetch the rate immediately at startup

	for {
		interval := time.Duration(rand.Intn(481)+120) * time.Second // Random interval between 120-600s
		time.Sleep(interval)
		updateRate()
	}
}

// updateRate performs a single fetch of the SOL-to-USD rate and updates the cache.
func updateRate() {
	log.Println("Fetching SOL-to-USD exchange rate...")
	response, err := http.Get("https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd")
	if err != nil {
		log.Printf("Failed to fetch SOL-to-USD rate: %v\n", err)
		return
	}
	defer response.Body.Close()

	var responseData map[string]map[string]float64
	if err := json.NewDecoder(response.Body).Decode(&responseData); err != nil {
		log.Printf("Failed to decode SOL-to-USD response: %v\n", err)
		return
	}

	rate, ok := responseData["solana"]["usd"]
	if !ok {
		log.Println("SOL-to-USD rate not found in response")
		return
	}

	interpreter.SetSolToUsdCache(rate) // Update the interpreter cache
	log.Printf("Updated SOL-to-USD rate in interpreter: %f\n", rate)
}

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
			time.Sleep(1 * time.Minute)
		}
	}()

	// Initialize Sniffer with webhook URL
	snifferInstance := sniffer.NewSniffer(walletManager, cfg.Application.WebhookURL)

	// Start the SOL-to-USD rate fetcher
	go fetchSolToUsdRate()

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
