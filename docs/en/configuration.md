# Configuration Guide

This document provides a detailed explanation of the configuration options for EVM Scanner.

## Configuration Files

EVM Scanner uses two main configuration files:

- **`config.yaml`** - Infrastructure configuration (RPC, Storage, Scanner parameters)
- **`app.yaml`** - Business logic configuration (Filters, Output destinations)

## config.yaml Details

### Basic Config

```yaml
# Unique project identifier
# Used to distinguish scanning progress in DB or Redis
project: "evm-scanner-service"
```

### Logging

```yaml
log:
  # Log level: debug, info, warn, error
  level: "info"
  
  # Log format:
  # - text: Dev mode, colored output, easy to read
  # - json: Production mode, structured output
  format: "text"
```

### Scanner Parameters

```yaml
scanner:
  # Chain identifier
  # Supported: ethereum, polygon, bsc, arbitrum, optimism, etc.
  chain_id: "ethereum"
  
  # === Start Strategy ===
  
  # Force start mode
  # true: Ignore saved progress, start from start_block
  # false: Resume from last saved progress (recommended)
  force_start: false
  
  # Starting block number
  # Only effective when force_start=true or for the first run
  start_block: 0
  
  # Initial rewind
  # On first run, start from (current block - start_rewind)
  # Helps avoid missing recent events
  start_rewind: 1000
  
  # Restart rewind
  # On restart, start from (last saved position - cursor_rewind)
  # Handles short-term chain reorganizations
  cursor_rewind: 10
  
  # === Performance ===
  
  # Batch size
  # Number of blocks to fetch in a single RPC request
  # Recommended: 10-100 (adjust based on RPC limits)
  batch_size: 50
  
  # Polling interval
  # Frequency to check for new blocks
  # Recommended: Chain block time / 2
  interval: "2s"
  
  # Confirmations
  # Scans up to (latest block - confirmations)
  # Avoids data inconsistency from chain reorgs
  # Ethereum recommendation: 12-64
  confirmations: 12
  
  # Bloom Filter
  # Enables node-level filtering for massive performance boost
  # Requires RPC node support
  use_bloom: true
  
  # Storage Prefix
  # Isolate data for different projects
  # Prepended to table names or Redis keys
  storage_prefix: "evm_scan_"
```

### RPC Node Pool

```yaml
# RPC Node Pool
# Supports high availability with automatic failover based on priority
rpc_nodes:
  # Primary node (paid tier, high performance)
  - url: "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
    priority: 10
    rate_limit: 25        # Max requests per second (QPS)
    max_concurrent: 10    # Max concurrent requests
  
  # Backup node (free tier)
  - url: "https://rpc.ankr.com/eth"
    priority: 5
    rate_limit: 10
    max_concurrent: 5
  
  # Backup node 2
  - url: "https://1rpc.io/eth"
    priority: 1
    rate_limit: 5
    max_concurrent: 3
```

**Parameter Details:**

- **url**: RPC endpoint address
- **priority**: Priority level (1-100), higher numbers = higher priority
- **rate_limit**: Per-node QPS limit to prevent hitting provider rate limits
  - 0 = unlimited (not recommended)
  - Set based on your RPC provider's limits
  - Alchemy/Infura paid: 25-50
  - Free public nodes: 5-10
- **max_concurrent**: Maximum concurrent requests per node
  - 0 = unlimited (not recommended)
  - Prevents node overload
  - Recommended: 30-50% of rate_limit

**Node Selection Mechanism:**
- Prioritizes high-priority nodes
- Automatically switches when nodes are busy or rate-limited
- Circuit breaker trips after 5 consecutive failures (30s timeout)
- Dynamic scoring based on latency, error rate, and block height
- Recommended: Configure 2-3 nodes for high availability

## app.yaml Details

### Filters

```yaml
filters:
  - description: "USDT Transfer Events"
    
    # Contract addresses
    # Leave empty to listen to all contracts
    contracts:
      - "0xdAC17F958D2ee523a2206206994597C13D831ec7"
    
    # Event Topics
    # Topic0 is the keccak256 hash of the event signature
    # Transfer(address,address,uint256)
    topics:
      - ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"]
    
    # ABI Definition (Optional)
    # Automatically decodes logs into human-readable format
    abi: '[{"anonymous":false,"inputs":[...],"name":"Transfer","type":"event"}]'
```

### Outputs

#### 1. Webhook

```yaml
outputs:
  webhook:
    enabled: true
    url: "https://your-api.com/webhook"
    secret: "your-signing-secret"
    
    # Retry strategy
    retry:
      max_attempts: 3
      initial_backoff: "1s"
      max_backoff: "10s"
    
    # Async mode (recommended)
    async: true
    buffer_size: 2000
    workers: 5
```

#### 2. PostgreSQL

```yaml
outputs:
  postgres:
    enabled: true
    url: "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
    table: "contract_events"
```

#### 3. Redis

```yaml
outputs:
  redis:
    enabled: true
    addr: "localhost:6379"
    password: ""
    db: 0
    key: "evm_events_queue"
    mode: "list"
```

## Best Practices

1. **Production**: Use structured `json` logs, multiple RPC nodes, and conservative `confirmations`.
2. **Performance**: Adjust `batch_size` based on RPC rate limits (usually 50-100).
3. **Security**: Use environment variables for sensitive info like API keys and DB credentials.
