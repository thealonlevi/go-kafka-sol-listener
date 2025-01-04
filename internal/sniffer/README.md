# Sniffer Module

The **Sniffer** module is a key component in the `go-kafka-sol-listener` project, responsible for processing messages received from Kafka, identifying relevant messages based on wallet activity, and forwarding them to a specified webhook for further processing.

## Features

- **Wallet-Based Filtering**: Matches messages containing specific wallet addresses against a dynamically managed list.
- **Concurrent Processing**: Handles matched messages asynchronously to improve throughput and responsiveness.
- **Latency Metrics**: Measures and logs processing and Kafka server latencies for performance monitoring.
- **Webhook Integration**: Sends matched messages to a configurable webhook for further analysis or action.

---

## Code Overview

### 1. **Sniffer Initialization**
```go
func NewSniffer(walletManager *wallet.WalletManager, webhookURL string) *Sniffer
```
Creates and initializes a new Sniffer instance with the following parameters:
- `walletManager`: Manages the list of wallets to match.
- `webhookURL`: The endpoint where matched messages are sent.

---

### 2. **Message Handling**
```go
func (s *Sniffer) HandleMessages(messages []map[string]interface{})
```
Processes a batch of Kafka messages:
- Sorts messages by their `Block.Timestamp`.
- Checks if any message contains a wallet address present in the wallet list.
- Sends matched messages to the webhook and logs relevant metrics.

---

### 3. **Swap Detection**
```go
func (s *Sniffer) processWithInterpreter(message map[string]interface{})
```
Forwards matched messages to the interpreter module for swap detection.

---

### 4. **Latency Metrics**
```go
func (s *Sniffer) logMetrics(timestampStart, timestampEnd, timestamp1 int64)
```
Logs key latency metrics:
- **Sniffer Latency**: Time taken to process a batch of messages.
- **Kafka Server Latency**: Time difference between the first message's timestamp and the end of processing.

---

### 5. **Webhook Communication**
```go
func (s *Sniffer) sendMatchedMessages(messages []map[string]interface{})
```
Sends matched messages to the configured webhook:
- Converts messages to JSON format.
- Uses HTTP POST requests to send the data.
- Logs errors and HTTP response statuses.

---

### 6. **Block Timestamp Extraction**
```go
func getBlockTimestamp(message map[string]interface{}) (int64, bool)
```
Safely extracts the `Block.Timestamp` field from a message, converting it to an integer.

---

## Configuration

The Sniffer module relies on the following configuration:
- **Wallet Manager**: Dynamically updates the list of wallet addresses to monitor.
- **Webhook URL**: Set during initialization, determines where matched messages are sent.

---

## How It Works

1. **Message Input**:
   The `HandleMessages` function receives a batch of messages.
2. **Sorting**:
   Messages are sorted chronologically by `Block.Timestamp`.
3. **Matching**:
   Each message is checked for wallet activity.
4. **Processing**:
   Matched messages are forwarded to the interpreter and sent to the webhook.
5. **Metrics Logging**:
   Latencies are calculated and logged for performance monitoring.

---

## Example Usage

### Sniffer Initialization
```go
walletManager := wallet.NewWalletManager("https://example.com/getWalletList")
webhookURL := "https://example.com/webhook"
sniffer := NewSniffer(walletManager, webhookURL)
```

### Handling Messages
```go
messages := []map[string]interface{}{
    {"Transaction": {"Signer": "wallet_address"}, "Block": {"Timestamp": 1631234567}},
}
sniffer.HandleMessages(messages)
```

---

## Logging and Monitoring

- Logs include:
  - Processing latencies.
  - Kafka server latencies.
  - Successful matches and webhook responses.
- Ensure logging is configured appropriately to capture important metrics.

---

## Future Improvements

- **Batch Optimization**: Enhance sorting and matching for larger batches.
- **Error Recovery**: Add retries for failed webhook requests.
- **Dynamic Configuration**: Allow runtime updates to the webhook URL and wallet list.

---

## Dependencies

- **Wallet Manager**: Manages wallet address updates.
- **Interpreter Module**: Processes messages for swap detection.
- **HTTP**: Used for webhook communication.

---

## License

This project is licensed under the MIT License.

