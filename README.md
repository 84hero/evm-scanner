# EVM Scanner 🚀

[![Go Report Card](https://goreportcard.com/badge/github.com/84hero/evm-scanner)](https://goreportcard.com/report/github.com/84hero/evm-scanner)
[![Build Status](https://github.com/84hero/evm-scanner/workflows/Test%20and%20Lint/badge.svg)](https://github.com/84hero/evm-scanner/actions)
[![GoDoc](https://godoc.org/github.com/84hero/evm-scanner?status.svg)](https://godoc.org/github.com/84hero/evm-scanner)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**[English](README.md)** | **[简体中文](README_CN.md)**

A high-performance, industrial-grade EVM event scanning and indexing framework. Built for developers who need reliable, real-time access to blockchain data without the overhead of complex indexing solutions.

[Features](#-features) • [Installation](#-installation) • [Quick Start](#-quick-start) • [Documentation](#-documentation) • [Contributing](#-contributing)

---

## 🌟 Features

-   **⛓️ Multi-Chain Native**: Optimized for Ethereum, BSC, Polygon, Arbitrum, and any EVM-compatible network.
-   **💾 Pluggable Storage**: Choose your persistence layer—**Memory** (dev), **Redis** (performance), or **PostgreSQL** (durability).
-   **🚀 High Performance**: 
    -   **Batch Processing**: Efficient RPC call batching to minimize latency and costs.
    -   **Bloom Filter Support**: Leverages node-level filtering for massive speed gains.
    -   **Worker Pool**: Parallel output processing (sinks) for high-throughput environments.
-   **🔌 Rich Ecosystem (Sinks)**: Stream data directly to **Webhooks**, **Kafka**, **RabbitMQ**, **Redis**, **PostgreSQL**, or flat files.
-   **🛡️ Production Ready**: Automatic reorg handling with configurable safety windows and cursor management.
-   **💎 Human Readable**: Built-in ABI decoding turns raw hex logs into structured JSON data automatically.

## 📦 Installation

### Binary (Recommended)
Download the pre-compiled binary for your architecture from the [Releases](https://github.com/84hero/evm-scanner/releases) page.

### Using Go
```bash
go install github.com/84hero/evm-scanner/cmd/scanner-cli@latest
```

### From Source
```bash
git clone https://github.com/84hero/evm-scanner.git
cd evm-scanner
make build
```

## 🚀 Quick Start

### 1. Initialize Configuration
```bash
cp config.yaml.example config.yaml
cp app.yaml.example app.yaml
```

### 2. Run the CLI
```bash
# Start scanning based on app.yaml filters
./bin/scanner-cli
```

### 3. Docker (One-Liner)
```bash
docker-compose up -d
```

## 📖 Documentation

Check out the detailed documentation for configuration and usage depth:

- [**Quick Start**](./docs/en/quick-start.md) - Get your first scanner running in 5 minutes.
- [**Configuration**](./docs/en/configuration.md) - Detailed guide for `config.yaml` and `app.yaml`.
- [**Architecture**](./docs/en/architecture.md) - Understand how EVM Scanner works under the hood.
- [**API Reference**](./docs/en/api-reference.md) - Webhook formats, CLI flags, and Database schema.
- [**Deployment**](./docs/en/deployment.md) - Production best practices and deployment strategies.
- [**Custom Sinks**](./docs/en/custom-sink.md) - Learn how to extend the output destinations.
- [**FAQ**](./docs/en/faq.md) - Frequently asked questions and common troubleshooting.


## 🛠 Usage Examples

### CLI Mode (Standalone)
Define your filters in `app.yaml`:
```yaml
filters:
  - description: "USDT Transfer Tracker"
    contracts: ["0xdAC17F958D2ee523a2206206994597C13D831ec7"]
    topics: ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"]
    abi: '[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},...],"name":"Transfer","type":"event"}]'
```

### SDK Mode (As a Library)
Explore our curated examples to see how to integrate `evm-scanner` into your stack:

| Example | Description |
| :--- | :--- |
| [**Basic SDK**](./examples/basic) | Minimal setup to start scanning from a Go app. |
| [**Custom Decoder**](./examples/custom-decoder) | How to decode raw logs into human-readable data using ABIs. |
| [**PostgreSQL Integration**](./examples/postgres-integration) | Production-ready setup using Postgres for both progress tracking and data storage. |
| [**Enterprise MQ**](./examples/enterprise-mq) | Streaming event data to **Kafka** for high-throughput microservices. |
| [**Multi-Sink Pipeline**](./examples/multi-sink) | Dispatching events to Console and Files simultaneously. |
| [**Custom Chain Preset**](./examples/custom-chain) | Configure parameters for a new L2 or AppChain (BlockTime, ReorgSafe). |
| [**Custom Sink**](./examples/custom-sink) | Extend the framework by implementing your own output destination (e.g., Slack). |
| [**Webhook Receiver**](./examples/webhook-receiver) | A simple server to receive and process events via Webhook. |

```go
import (
    "github.com/84hero/evm-scanner/pkg/scanner"
    "github.com/84hero/evm-scanner/pkg/rpc"
)

func main() {
    client, _ := rpc.NewClient(ctx, rpcCfg, 10)
    s := scanner.New(client, storage, scanCfg, filter)
    
    s.SetHandler(func(ctx context.Context, logs []types.Log) error {
        // Your custom business logic here
        return nil
    })
    
    s.Start(ctx)
}
```

## ⚙️ Configuration

The project uses two primary configuration files:
| File | Purpose | Key Settings |
| :--- | :--- | :--- |
| `config.yaml` | Infrastructure | RPC Nodes, DB/Redis connections, Scan speed |
| `app.yaml` | Business Logic | Contracts, Topics, ABI, Output Destinations |

## 🏗 Supported Sinks (Outputs)

| Sink | Status | Use Case |
| :--- | :--- | :--- |
| **Webhook** | ✅ | Real-time API integration |
| **PostgreSQL** | ✅ | Permanent event storage & querying |
| **Redis** | ✅ | Fast message passing (List/PubSub) |
| **Kafka** | ✅ | Big data pipelines & stream processing |
| **RabbitMQ** | ✅ | Enterprise message queuing |
| **Console/File** | ✅ | Debugging and logging |

## 🛠 Development

We use `Makefile` for common tasks:
- `make test`: Run the test suite.
- `make lint`: Run code quality checks.
- `make snapshot`: Local build validation with GoReleaser.

## 🤝 Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📧 Contact

- **Telegram**: [@xNew4](https://t.me/xNew4)
- **X (Twitter)**: [@0xNew4](https://x.com/0xNew4)

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.

---
Built with ❤️ for the Web3 Community.