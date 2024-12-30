# Go-Kafka-Sol-Listener

## Overview

Go-Kafka-Sol-Listener is a high-performance application designed to process Solana blockchain transaction data from a Kafka stream. The application identifies transactions involving specific wallets, calculates performance metrics, and sends relevant data to a specified webhook.

---

## Features

1. **Kafka Consumer:**
   - Subscribes to a Kafka topic and processes batches of messages.

2. **Wallet Matching:**
   - Matches transactions based on a dynamic list of wallet addresses fetched from an external API.

3. **Sniffer Metrics:**
   - Calculates latency metrics including:
     - Sniffer Latency
     - Batch Latency
     - Consumer Latency (1, 2, 3)
     - Total Latency

4. **Webhook Integration:**
   - Sends matched transactions and performance data to a specified webhook.

5. **Message Sorting:**
   - Ensures all incoming messages are sorted by block timestamp for consistent processing.

---

## Project Directory Structure

```
go-kafka-sol-listener/
├── cmd/
│   └── main.go         # Entry point for the application
├── config/
│   └── config.yaml     # Configuration file for the application
├── env/
│   ├── client.cer.pem  # Client certificate
│   ├── client.key.pem  # Client key
│   └── server.cer.pem  # Server certificate
├── internal/
│   ├── config/
│   │   └── config_loader.go  # Configuration loader
│   ├── consumer/
│   │   └── consumer.go       # Kafka consumer logic
│   ├── sniffer/
│   │   └── sniffer.go        # Sniffing and processing Solana transactions
│   └── wallet/
│       └── wallet.go         # Wallet management logic
├── test/
│   └── test.go         # Test script for local debugging
├── .gitignore           # Git ignore file
├── go.mod               # Go module dependencies
├── go.sum               # Dependency checksums
└── README.md            # Project documentation
```

---

## Requirements

### Software
- Go (1.23.3 or later)
- librdkafka (for Confluent Kafka Go library)

### Libraries
- `github.com/confluentinc/confluent-kafka-go` (v1.9.2)
- `gopkg.in/yaml.v2` (v2.4.0)

---

## Installation

### 1. Clone the Repository
```bash
$ git clone https://github.com/thealonlevi/go-kafka-sol-listener.git
$ cd go-kafka-sol-listener
```

### 2. Install Dependencies
```bash
$ go mod tidy
```

### 3. Install librdkafka
#### Ubuntu
```bash
$ sudo apt-get update
$ sudo apt-get install -y librdkafka-dev
```
#### MacOS
```bash
$ brew install librdkafka
```

---

## Configuration

The application configuration is managed via a `config.yaml` file located in the `config/` directory. Ensure the file contains the following fields:

```yaml
kafka:
  bootstrap_servers:
    - "localhost:9092"
  group_id: "consumer-group"
  topic: "transactions"
  security:
    protocol: "plaintext"
    sasl_mechanisms: ""
    username: ""
    password: ""
    ssl_ca_location: ""
    ssl_key_location: ""
    ssl_certificate_location: ""
    endpoint_identification_algorithm: ""
  auto_offset_reset: "earliest"

webhook_url: "https://example.com/webhook"
```

---

## Running the Application

### 1. Start the Consumer
```bash
$ go run cmd/main.go
```

### 2. Using TMUX for Multiple Instances
To run multiple instances of the consumer:

- Start TMUX session:
```bash
$ tmux new-session -s kafka-listener
```
- Start the application within TMUX:
```bash
$ go run cmd/main.go
```
- Detach from the session:
```bash
Ctrl+b, then d
```
- List TMUX sessions:
```bash
$ tmux list-sessions
```
- Reattach to a session:
```bash
$ tmux attach-session -t kafka-listener
```

---

## Metrics Calculated

### Latency Metrics
- **Sniffer Latency:** Time taken by the sniffer to process a batch.
- **Batch Latency:** Difference between the timestamps of the first and last message in the batch.
- **Consumer Latency:** Measures the delay in processing messages at three points (start, middle, end).
- **Total Latency:** End-to-end latency for the batch.

### Example Output:
```plaintext
Sniffer Latency: 3 seconds
Batch Latency: 10 seconds
ConsumerLatency1: 5 seconds
ConsumerLatency2: 6 seconds
ConsumerLatency3: 8 seconds
Total Latency: 11 seconds
```

---

## Development Notes

### Sorting Messages
The first step in processing is sorting all incoming messages by the `Block.Timestamp` field (earliest to latest).

### Thread-Safe Operations
The application uses mutex locks to ensure thread safety when managing critical resources, such as sending matched messages.

### Logging
All significant operations are logged to the console for debugging and analysis.

---

## Contributing
Contributions are welcome! Please submit a pull request or open an issue for any bugs or feature requests.

---

## License
This project is licensed under the MIT License. See the LICENSE file for details.

---

## Contact
For any questions or support, please contact Alon Levi at [levialon@proton.me](mailto:levialon@proton.me).

