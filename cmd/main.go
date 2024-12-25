package main

import (
	"fmt"
	"go-kafka-sol-listener/internal"
	"go-kafka-sol-listener/internal/config"
	"log"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	fmt.Printf("Loaded configuration: %+v\n", cfg)

	// Start Kafka consumer
	fmt.Println("Starting Kafka consumer...")
	if err := internal.StartConsumer(cfg); err != nil {
		log.Fatalf("Failed to start Kafka consumer: %v", err)
	}
}
