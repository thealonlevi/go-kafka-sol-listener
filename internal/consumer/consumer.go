package consumer

import (
	"encoding/json"
	"fmt"
	"go-kafka-sol-listener/internal/config"
	"go-kafka-sol-listener/internal/sniffer"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// StartConsumer initializes the Kafka consumer and processes messages.
// It takes in the application configuration and the sniffer instance for processing messages.
func StartConsumer(cfg *config.Config, sniffer *sniffer.Sniffer) error {
	// Step 1: Construct the Kafka bootstrap servers string from the configuration.
	bootstrapServers := ""
	for i, server := range cfg.Kafka.BootstrapServers {
		bootstrapServers += server
		if i < len(cfg.Kafka.BootstrapServers)-1 {
			bootstrapServers += "," // Add a comma separator between servers.
		}
	}

	// Step 2: Create the Kafka consumer using the configuration provided.
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":                     bootstrapServers,                                   // List of Kafka brokers.
		"group.id":                              cfg.Kafka.GroupID,                                  // Consumer group ID.
		"security.protocol":                     cfg.Kafka.Security.Protocol,                        // Security protocol.
		"sasl.mechanisms":                       cfg.Kafka.Security.SASLMechanisms,                  // SASL authentication mechanism.
		"sasl.username":                         cfg.Kafka.Security.Username,                        // SASL username.
		"sasl.password":                         cfg.Kafka.Security.Password,                        // SASL password.
		"ssl.ca.location":                       cfg.Kafka.Security.SSLCALocation,                   // Path to CA certificate.
		"ssl.key.location":                      cfg.Kafka.Security.SSLKeyLocation,                  // Path to SSL key.
		"ssl.certificate.location":              cfg.Kafka.Security.SSLCertificateLocation,          // Path to SSL certificate.
		"ssl.endpoint.identification.algorithm": cfg.Kafka.Security.EndpointIdentificationAlgorithm, // Endpoint verification.
		"auto.offset.reset":                     cfg.Kafka.AutoOffsetReset,                          // Where to start reading messages.
	})
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err) // Handle consumer creation failure.
	}
	defer consumer.Close() // Ensure the consumer is closed when the function exits.

	// Step 3: Subscribe to the Kafka topic specified in the configuration.
	err = consumer.SubscribeTopics([]string{cfg.Kafka.Topic}, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err) // Handle subscription failure.
	}

	// Step 4: Determine the polling interval from configuration (in milliseconds).
	pollInterval := 1000 // Default value is 1000ms if not configured.
	if cfg.Kafka.PollIntervalMs > 0 {
		pollInterval = cfg.Kafka.PollIntervalMs // Use configured value if available.
	}

	fmt.Println("Consumer is now listening for messages...")

	// Step 5: Main loop to continuously poll for Kafka messages.
	for {
		ev := consumer.Poll(pollInterval) // Poll for messages/events from Kafka.
		if ev == nil {
			continue // If no event, continue to the next poll cycle.
		}

		switch e := ev.(type) {
		case *kafka.Message: // Handle Kafka message event.
			if err := handleKafkaMessage(e, sniffer); err != nil {
				log.Printf("Failed to handle Kafka message: %v", err) // Log any message handling errors.
			}
		case kafka.Error: // Handle Kafka error event.
			log.Printf("Kafka error: %v", e)
			if e.IsFatal() {
				return fmt.Errorf("fatal Kafka error: %w", e) // Exit on fatal errors.
			}
		default: // Handle any other events.
			log.Printf("Ignored event: %v", e)
		}
	}
}

// handleKafkaMessage processes a single Kafka message and sends it to the sniffer for processing.
func handleKafkaMessage(msg *kafka.Message, sniffer *sniffer.Sniffer) error {
	// Step 1: Parse the message value into a slice of maps.
	var messages []map[string]interface{}
	if err := json.Unmarshal(msg.Value, &messages); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err) // Handle JSON parsing errors.
	}

	// Step 2: Pass the parsed messages to the sniffer for further processing.
	sniffer.HandleMessages(messages)
	return nil // Indicate successful handling of the message.
}
