package main

import (
	"encoding/json"
	"fmt"
	"go-kafka-sol-listener/internal/config"
	"go-kafka-sol-listener/internal/consumer"
	"go-kafka-sol-listener/internal/interpreter"
	"go-kafka-sol-listener/internal/metrics"
	"go-kafka-sol-listener/internal/sniffer"
	"go-kafka-sol-listener/internal/utils"
	"go-kafka-sol-listener/internal/wallet"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var instanceUIDCache struct {
	uid   string
	mutex sync.RWMutex
}

var solToUsdAPIURL string
var uidCharset string
var metricsFlushInterval int
var metricsEndpoint string

func generateInstanceUID() string {
	uid := make([]byte, 16)
	for i := range uid {
		uid[i] = uidCharset[rand.Intn(len(uidCharset))]
	}
	return string(uid)
}

func setInstanceUIDCache(uid string) {
	instanceUIDCache.mutex.Lock()
	defer instanceUIDCache.mutex.Unlock()
	instanceUIDCache.uid = uid
}

func getInstanceUIDCache() string {
	instanceUIDCache.mutex.RLock()
	defer instanceUIDCache.mutex.RUnlock()
	return instanceUIDCache.uid
}

func logInstanceUID() {
	for {
		log.Printf("Instance UID: %s\n", getInstanceUIDCache())
		time.Sleep(30 * time.Second)
	}
}

func fetchSolToUsdRate() {
	updateRate()

	for {
		interval := time.Duration(rand.Intn(481)+120) * time.Second
		time.Sleep(interval)
		updateRate()
	}
}

func updateRate() {
	log.Println("Fetching SOL-to-USD exchange rate...")
	response, err := http.Get(solToUsdAPIURL)
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

	interpreter.SetSolToUsdCache(rate)
	log.Printf("Updated SOL-to-USD rate: %.2f\n", rate)
}

func initSystem(cfg *config.Config) {
	uidCharset = cfg.Application.UIDCharset
	solToUsdAPIURL = cfg.Application.SolToUsdAPIURL
	metricsFlushInterval = cfg.Metrics.FlushIntervalSeconds
	metricsEndpoint = cfg.Metrics.CloudEndpoint

	instanceUID := generateInstanceUID()
	setInstanceUIDCache(instanceUID)
	utils.SetInstanceUID(instanceUID)

	log.Printf("Instance UID set to: %s\n", instanceUID)
}

func startMetricsHandler(metricsHandler *metrics.MetricsHandler) {
	ticker := time.NewTicker(time.Duration(metricsFlushInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		totalMessages, avgDelay := metricsHandler.AggregateAndClear()

		payload := map[string]interface{}{
			"instanceUID":    getInstanceUIDCache(),
			"messagesPerMin": totalMessages,
			"avgDelay":       avgDelay,
		}

		log.Printf("Reporting metrics: %+v\n", payload)

		err := metrics.SendMetrics(metricsEndpoint, payload)
		if err != nil {
			log.Printf("Failed to report metrics: %v\n", err)
		}
	}
}

func main() {
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	initSystem(cfg)
	interpreter.InitializeInterpreterConfig(cfg)

	go logInstanceUID()
	go fetchSolToUsdRate()

	walletManager := wallet.NewWalletManager(cfg.Application.WalletListURL, cfg.Application.WalletUpdateInterval)
	go walletManager.UpdateWallets()

	go func() {
		for {
			wallets := walletManager.GetWalletList()
			log.Printf("Current wallet list: %v\n", wallets)
			time.Sleep(1 * time.Minute)
		}
	}()

	metricsHandler := metrics.GetMetricsHandler(getInstanceUIDCache())
	go startMetricsHandler(metricsHandler)

	snifferInstance := sniffer.NewSniffer(walletManager, cfg.Application.WebhookURL, metricsHandler, cfg.Application.SaveMatches, cfg.Application.TransferWebhookURL)

	for {
		fmt.Println("Starting Kafka consumer...")
		err := consumer.StartConsumer(cfg, snifferInstance)
		if err != nil {
			log.Printf("Kafka consumer encountered an error: %v. Restarting in 5 seconds...", err)
			time.Sleep(5 * time.Second)
		}
	}
}
