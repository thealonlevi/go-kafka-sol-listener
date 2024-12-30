# Go Kafka Sol Listener

Go Kafka Sol Listener is a robust application designed to listen to Kafka messages, process Solana blockchain transactions, and integrate with webhooks for real-time updates. This project is highly scalable, efficient, and well-suited for handling large volumes of transactions.

## Features

- **Real-Time Transaction Processing**: Listens to Kafka topics for Solana blockchain transactions.
- **In-Memory Cache**: Utilizes caching for optimized performance.
- **Asynchronous Processing**: Handles multiple tasks simultaneously for efficient message consumption.
- **Webhook Integration**: Sends matched transactions to configured webhooks.
- **Dynamic Scaling**: Designed to handle high throughput with ease.

## Architecture

The project is modular and follows a well-defined structure:

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
├── .gitignore           # Git ignore file
├── go.mod               # Go module dependencies
├── go.sum               # Dependency checksums
```

### Core Components

1. **Listener**:
   - Connects to Kafka topics and listens for new messages.
   - Maintains a persistent WebSocket connection.

2. **Processor**:
   - Processes Solana transactions received from Kafka.
   - Matches transactions against a list of wallets stored in DynamoDB.

3. **Consumer**:
   - Manages message consumption from Kafka topics.
   - Ensures robust error handling and retries.

4. **Sniffer**:
   - Implements the logic for decoding and analyzing Solana transactions.

5. **Wallet Management**:
   - Handles wallet-related operations, such as fetching and updating wallet lists.

6. **Configuration Management**:
   - Centralized configuration loading through `config_loader.go`.

7. **Certificates and Security**:
   - TLS certificates stored in the `env` folder for secure communication.

## Setup and Configuration

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/yourusername/go-kafka-sol-listener.git
   cd go-kafka-sol-listener
   ```

2. **Install Dependencies**:
   Ensure you have Go installed. Run:
   ```bash
   go mod tidy
   ```

3. **Configure the Application**:
   Update the `config/config.yaml` file with the following:
   - Kafka Broker URL
   - Solana WebSocket API URL
   - Webhook URL
   - AWS Lambda Endpoint
   - Cache Update Interval

4. **Run the Application**:
   ```bash
   go run cmd/main.go
   ```

## Usage

1. Start the listener to begin processing Kafka messages.
2. Monitor matched transactions being sent to the webhook endpoint.
3. Use logs to debug or analyze real-time processing.

## Future Enhancements

- **Batching Webhook Requests**: Improve performance by grouping transactions into batches.
- **Bloom Filters**: Optimize cache performance with probabilistic data structures.
- **Dynamic Configuration Reloading**: Allow runtime updates to the configuration file.

## Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature or bug fix.
3. Submit a pull request with a detailed description.

## License

This project is licensed under the [MIT License](LICENSE).

## Contact

For questions or support, please reach out to the project maintainer:

- **Name**: Alon Levi
- **Email**: [levialon@proton.me](mailto:levialon@proton.me)
