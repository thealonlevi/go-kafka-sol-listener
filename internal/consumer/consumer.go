package consumer

import (
	"encoding/json"
	"fmt"
	"go-kafka-sol-listener/internal/config"
	"go-kafka-sol-listener/internal/sniffer"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func StartConsumer(cfg *config.Config, sniffer *sniffer.Sniffer) error {
	// Join bootstrap servers into a single string
	bootstrapServers := ""
	for i, server := range cfg.Kafka.BootstrapServers {
		bootstrapServers += server
		if i < len(cfg.Kafka.BootstrapServers)-1 {
			bootstrapServers += ","
		}
	}

	// Create Kafka consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":                     bootstrapServers,
		"group.id":                              cfg.Kafka.GroupID,
		"security.protocol":                     cfg.Kafka.Security.Protocol,
		"sasl.mechanisms":                       cfg.Kafka.Security.SASLMechanisms,
		"sasl.username":                         cfg.Kafka.Security.Username,
		"sasl.password":                         cfg.Kafka.Security.Password,
		"ssl.ca.location":                       cfg.Kafka.Security.SSLCALocation,
		"ssl.key.location":                      cfg.Kafka.Security.SSLKeyLocation,
		"ssl.certificate.location":              cfg.Kafka.Security.SSLCertificateLocation,
		"ssl.endpoint.identification.algorithm": cfg.Kafka.Security.EndpointIdentificationAlgorithm,
		"auto.offset.reset":                     cfg.Kafka.AutoOffsetReset,
	})
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}
	defer consumer.Close()

	// Subscribe to Kafka topic
	err = consumer.SubscribeTopics([]string{cfg.Kafka.Topic}, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	// Consume messages and offload to sniffer every 500ms
	fmt.Println("Consumer is now listening for messages...")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var batchedMessages []map[string]interface{}

	go func() {
		for range ticker.C {
			if len(batchedMessages) > 0 {
				log.Printf("Passing %d messages to sniffer", len(batchedMessages))
				go sniffer.HandleMessages(batchedMessages)
				batchedMessages = []map[string]interface{}{} // Reset batched messages
			}
		}
	}()

	for {
		ev := consumer.Poll(100)
		if ev == nil {
			continue
		}

		switch e := ev.(type) {
		case *kafka.Message:
			var messages []map[string]interface{} // Updated to handle arrays of messages
			if err := json.Unmarshal(e.Value, &messages); err != nil {
				log.Printf("Failed to unmarshal message: %v\n", err)
				continue
			}
			// Add all messages from the array to the batch
			for _, message := range messages {
				batchedMessages = append(batchedMessages, message)
			}
		case kafka.Error:
			log.Printf("Kafka error: %v\n", e)
		}
	}
}
