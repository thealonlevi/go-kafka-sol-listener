# Kafka Consumer for Solana Balance Updates

This Kafka consumer is responsible for consuming Solana balance updates from a Kafka topic and processing them through the application pipeline. The consumer leverages Confluent Kafka and integrates with a `Sniffer` to handle messages.

## Features

- **Poll-based Message Processing**: Messages are polled from Kafka using a configurable interval, eliminating the need for manual batching.
- **Secure Kafka Connection**: Supports SASL_SSL protocol with required credentials and certificates.
- **Dynamic Configuration**: Customizable Kafka settings through `config.yaml`.
- **Integration with Sniffer**: Automatically forwards messages to the `Sniffer` for processing.
- **Resilient Design**: Restarts on fatal errors with a delay.

## Prerequisites

1. **Kafka Cluster**: Ensure a Kafka cluster is set up and reachable.
2. **Configuration File**: A `config.yaml` file must be present with the required settings.
3. **Dependencies**:
   - `github.com/confluentinc/confluent-kafka-go/kafka`
   - Other Go modules as specified in `go.mod`.

## Configuration

The `config.yaml` file should include the following structure:

```yaml
kafka:
  bootstrap_servers:
    - "rpk0.bitquery.io:9093"
    - "rpk1.bitquery.io:9093"
    - "rpk2.bitquery.io:9093"
  group_id: "solanatest3-group-96"
  topic: "solana.balance_updates"
  security:
    protocol: "SASL_SSL"
    sasl_mechanisms: "SCRAM-SHA-512"
    username: "solanatest3"
    password: "<your_password>"
    ssl_ca_location: "env/server.cer.pem"
    ssl_key_location: "env/client.key.pem"
    ssl_certificate_location: "env/client.cer.pem"
    endpoint_identification_algorithm: "none"
  auto_offset_reset: "latest"
  poll_interval_ms: 1000  # Configurable polling interval in milliseconds
```

## Usage

1. **Initialize the Kafka Consumer**:
   The consumer is started by calling the `StartConsumer` function with the configuration and an instance of `Sniffer`.

   ```go
   cfg, err := config.LoadConfig("config/config.yaml")
   if err != nil {
       log.Fatalf("Failed to load configuration: %v", err)
   }

   snifferInstance := sniffer.NewSniffer(walletManager, "<webhook_url>")

   err = consumer.StartConsumer(cfg, snifferInstance)
   if err != nil {
       log.Fatalf("Failed to start consumer: %v", err)
   }
   ```

2. **Run the Application**:
   Compile and run the Go application:

   ```bash
   go run main.go
   ```

## Code Walkthrough

### Main Functionality
- **Consumer Initialization**: Establishes a Kafka consumer using credentials and settings from `config.yaml`.
- **Polling**: The consumer continuously polls messages from the configured Kafka topic at intervals defined by `poll_interval_ms`.
- **Message Processing**: Each polled message is directly forwarded to the `Sniffer` for processing.

### Key Configurations
- `poll_interval_ms`: Controls the frequency of message polling.
- `auto_offset_reset`: Determines where to start consuming messages when no offset is saved (e.g., `latest`, `earliest`).

## Error Handling

- **Non-fatal Errors**: Logged and ignored to ensure continuous operation.
- **Fatal Errors**: Triggers a restart with a delay to ensure resilience.

## Contribution

Feel free to contribute by creating pull requests or raising issues for enhancements and bug fixes.

## License

This project is licensed under the [MIT License](LICENSE).

---

For further details, refer to the `consumer.go` file or the main application code.
