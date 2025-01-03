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

// StartConsumer initializes the Kafka consumer and processes messages.
func StartConsumer(cfg *config.Config, sniffer *sniffer.Sniffer) error {
	// Join bootstrap servers into a single string.
	bootstrapServers := ""
	for i, server := range cfg.Kafka.BootstrapServers {
		bootstrapServers += server
		if i < len(cfg.Kafka.BootstrapServers)-1 {
			bootstrapServers += ","
		}
	}

	// Create Kafka consumer.
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

	// Subscribe to Kafka topic.
	err = consumer.SubscribeTopics([]string{cfg.Kafka.Topic}, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	// Consume messages and offload to sniffer every 1000ms.
	fmt.Println("Consumer is now listening for messages...")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var batchedMessages []map[string]interface{}

	// Goroutine to periodically process messages.
	go func() {
		for range ticker.C {
			if len(batchedMessages) > 0 {
				messagesToProcess := batchedMessages
				batchedMessages = nil // Reset the batch.
				go sniffer.HandleMessages(messagesToProcess)
			}
		}
	}()

	// Main loop to poll Kafka messages.
	for {
		ev := consumer.Poll(100)
		if ev == nil {
			continue
		}

		switch e := ev.(type) {
		case *kafka.Message:
			if err := handleKafkaMessage(e, &batchedMessages); err != nil {
				log.Printf("Failed to handle Kafka message: %v", err)
			}
		case kafka.Error:
			log.Printf("Kafka error: %v", e)
			if e.IsFatal() {
				return fmt.Errorf("fatal Kafka error: %w", e)
			}
		default:
			log.Printf("Ignored event: %v", e)
		}
	}
}

// handleKafkaMessage processes a Kafka message and appends it to the batch.
func handleKafkaMessage(msg *kafka.Message, batch *[]map[string]interface{}) error {
	var messages []map[string]interface{}
	if err := json.Unmarshal(msg.Value, &messages); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}
	*batch = append(*batch, messages...)
	return nil
}
