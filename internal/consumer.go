package internal

import (
	"fmt"
	"go-kafka-sol-listener/internal/config"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func StartConsumer(cfg *config.Config) error {
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

	// Consume messages
	fmt.Println("Consumer is now listening for messages...")
	for {
		ev := consumer.Poll(100)
		if ev == nil {
			continue
		}

		switch e := ev.(type) {
		case *kafka.Message:
			log.Printf("Message received: %s [Partition: %d, Offset: %d]\n",
				string(e.Value), e.TopicPartition.Partition, e.TopicPartition.Offset)
		case kafka.Error:
			log.Printf("Kafka error: %v\n", e)
		}
	}
}
