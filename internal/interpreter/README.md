# Interpreter Package - README

The `interpreter` package in the `go-kafka-sol-listener` project is responsible for processing, enriching, and forwarding Solana blockchain transactions. It integrates multiple components, such as BitQuery APIs, Python scripts, caching mechanisms, and webhook handlers, to ensure efficient transaction processing and seamless communication with external systems.

---

## Table of Contents
1. [Overview](#overview)
2. [Features](#features)
3. [Package Structure](#package-structure)
4. [Installation and Setup](#installation-and-setup)
5. [Usage](#usage)
6. [Detailed Components](#detailed-components)
   - [BitQuery Client](#bitquery-client)
   - [Cache Management](#cache-management)
   - [Enrichment Logic](#enrichment-logic)
   - [Python Executor](#python-executor)
   - [Signature Management](#signature-management)
   - [Webhook Handling](#webhook-handling)
7. [Configuration](#configuration)
8. [Examples](#examples)
9. [Testing](#testing)
10. [Contributing](#contributing)

---

## Overview
The `interpreter` package is a core module within the `go-kafka-sol-listener` project. It processes incoming Solana transactions, determines if they meet specific criteria (e.g., swaps), enriches the data with additional information (e.g., USD values, token supply), and forwards the results to a configured webhook. The package ensures efficient and secure transaction handling through caching, API integrations, and Python script execution.

---

## Features
- **Transaction Processing**: Parses and validates Solana transaction data.
- **Data Enrichment**: Adds SOL-to-USD conversion, token supply details, and additional metadata.
- **BitQuery Integration**: Fetches token details using BitQuery APIs.
- **Python Script Execution**: Executes external Python scripts for swap detection.
- **Caching**: Implements caching for frequently used data like SOL-to-USD rates.
- **Webhook Handling**: Sends enriched transaction data to an external webhook.
- **Signature Management**: Tracks processed signatures to prevent duplicate processing.

---

## Package Structure
```
interpreter/
├── bitquery_client.go           # Interacts with the BitQuery API
├── cache.go                     # Caching logic for SOL-to-USD rate
├── enrichment.go                # Enriches transactions with additional details
├── interpreter.go               # Core processing logic for transactions
├── logging.go                   # Handles logging for interpreter operations
├── python_executor.go           # Executes Python scripts for specific tasks
├── signature.go                 # Manages transaction signature validation and tracking
├── utils.go                     # General utility functions specific to the interpreter
├── webhook.go                   # Handles webhook operations
```

---

## Installation and Setup

1. **Dependencies**:
   Ensure the following dependencies are installed and available:
   - Go 1.18+
   - Python 3.x (required for swap detection script execution)

2. **Configuration**:
   Add the `interpreter` section in your `config.yaml` file:
   ```yaml
   interpreter:
     bitquery_token: "<your-bitquery-token>"
     python: "python"
     swap_detector_script: "scripts/swapdetector.py"
   ```

3. **Environment Setup**:
   - Install Python dependencies (if any) for the swap detection script:
     ```bash
     pip install -r scripts/requirements.txt
     ```

---

## Usage

### Initialization
Before using the `interpreter` package, initialize it with the required configuration:
```go
import (
    "go-kafka-sol-listener/internal/config"
    "go-kafka-sol-listener/internal/interpreter"
)

// Load configuration
cfg, err := config.LoadConfig("config/config.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Initialize the interpreter
interpreter.InitializeInterpreterConfig(cfg)
```

### Processing Transactions
Use the `ProcessMessage` function to handle transactions:
```go
import "go-kafka-sol-listener/internal/interpreter"

jsonData := []byte(`{"Transaction": {...}}`)
webhookURL := "https://example.com/webhook"

if err := interpreter.ProcessMessage(jsonData, webhookURL); err != nil {
    log.Printf("Error processing message: %v", err)
}
```

---

## Detailed Components

### BitQuery Client
File: `bitquery_client.go`

- Fetches token details (e.g., name, supply) from BitQuery APIs.
- Uses the `bitquery_token` from the configuration for authentication.

### Cache Management
File: `cache.go`

- Implements a thread-safe cache for storing SOL-to-USD exchange rates.
- Provides functions to set and get the cached rate.

### Enrichment Logic
File: `enrichment.go`

- Enriches transaction data with additional details, such as:
  - SOL-to-USD conversions.
  - Token supply details from BitQuery.
  - Fee calculations.
- Ensures correct token order for processing.

### Python Executor
File: `python_executor.go`

- Executes the Python swap detection script specified in the configuration.
- Handles input/output and logs errors.

### Signature Management
File: `signature.go`

- Tracks processed signatures to prevent duplicate processing.
- Uses in-memory caching for efficient lookup.

### Webhook Handling
File: `webhook.go`

- Sends enriched transaction data to a configured webhook.
- Supports error handling and retries.

---

## Configuration
Example `config.yaml`:
```yaml
interpreter:
  bitquery_token: "<your-bitquery-token>"
  python: "python"
  swap_detector_script: "scripts/swapdetector.py"
```

---

## Examples

### Fetching Token Supply
```go
name, supply, err := FetchTokenSupply("HkNokfCXG33eu5vCcS49mq3jZcKZeQSQCyta964YxxYg")
if err != nil {
    log.Printf("Failed to fetch token supply: %v", err)
} else {
    log.Printf("Token Name: %s, Supply: %s", name, supply)
}
```

### Enriching Transaction Details
```go
details := map[string]interface{}{
    "Token1": {"AmountChange": 1.0},
    "Token2": {"AmountChange": -10.0},
}
enrichSwapDetails(details, 182.5) // Assuming SOL-to-USD rate is 182.5
log.Printf("Enriched Details: %+v", details)
```

---

## Testing

1. **Unit Tests**:
   Run unit tests for the package:
   ```bash
   go test ./internal/interpreter/...
   ```

2. **Integration Tests**:
   Test end-to-end functionality with mock APIs and Python scripts.

3. **Manual Testing**:
   Use the `tester.go` script to simulate transactions and verify processing.

---

## Contributing

1. Fork the repository.
2. Create a feature branch.
3. Make your changes and add tests.
4. Submit a pull request with detailed descriptions of the changes.

---

This README provides a comprehensive overview of the `interpreter` package and its usage. For further assistance, please refer to the project's documentation or contact the maintainers.

