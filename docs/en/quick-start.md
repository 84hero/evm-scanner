# Quick Start

Get EVM Scanner up and running in less than 5 minutes.

## Prerequisites

- Go 1.21+ (if building from source)
- RPC node access for an EVM chain

## Installation

### 1. Pre-compiled Binary
Download from [Releases](https://github.com/84hero/evm-scanner/releases).

### 2. Using Go
```bash
go install github.com/84hero/evm-scanner/cmd/scanner-cli@latest
```

## Basic Setup

### 1. Initialize Config
```bash
cp config.yaml.example config.yaml
cp app.yaml.example app.yaml
```

### 2. Set RPC Node
Edit `config.yaml`:
```yaml
rpc_nodes:
  - url: "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
    priority: 10
```

### 3. Run
```bash
./scanner-cli
```

## Next Steps
- Read the [Configuration Guide](./configuration.md)
- Explore [Architecture](./architecture.md)
