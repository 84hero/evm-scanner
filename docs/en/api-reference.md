# API Reference

This document provides interface definitions, data formats, and CLI references for EVM Scanner.

## Command Line Interface (CLI)

The `scanner-cli` supports the following environment variables for configuration:

### Environment Variables
- `CONFIG_FILE`: Path to `config.yaml` (Default: `./config.yaml`)
- `APP_CONFIG_FILE`: Path to `app.yaml` (Default: `./app.yaml`)
- `PG_URL`: Connection string for Postgres storage (overrides config).
- `REDIS_ADDR`: Address for Redis storage (overrides config).

### Run Examples
```bash
# Start with default config
./scanner-cli

# Run with custom config paths
CONFIG_FILE=./prod/config.yaml APP_CONFIG_FILE=./prod/app.yaml ./scanner-cli
```

## Webhook Data Format

When the Webhook output is enabled, EVM Scanner sends a JSON `POST` request to the specified URL.

### Payload Structure
Logs are sent in batches:

```json
[
  {
    "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
    "topics": [
      "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
      "0x0000..."
    ],
    "data": "0x0000...",
    "blockNumber": 18000000,
    "transactionHash": "0xabc...",
    "transactionIndex": 42,
    "logIndex": 0,
    "eventName": "Transfer",
    "decoded": {
      "from": "0x123...",
      "to": "0x456...",
      "value": "100000000"
    }
  }
]
```

### Signature Verification
If a `secret` is configured, the request includes an `X-Scanner-Signature` header:
`HMAC-SHA256(payload_body, secret)`

## Database Schema (Postgres)

When using Postgres output, the application maintains the following table:

### `contract_events` Table

| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | SERIAL | Primary Key |
| `block_number` | BIGINT | Block height |
| `transaction_hash` | VARCHAR(66) | Tx Hash |
| `log_index` | INTEGER | Log index in block |
| `address` | VARCHAR(42) | Contract address |
| `topics` | JSONB | List of event topics |
| `data` | TEXT | Raw data |
| `decoded` | JSONB | Decoded parameters |
| `created_at` | TIMESTAMP | Injection time |

## Error Codes

| Code | Description | Recommended Action |
| :--- | :--- | :--- |
| `RPC_ERROR` | RPC node unreachable | Check network or add backup nodes |
| `DECODE_ERROR` | ABI mismatch or invalid data | Check ABI definition in `app.yaml` |
| `SINK_ERROR` | Downstream output failed | Check Webhook receiver or DB status |
| `REORG_DETECTED` | Chain reorganization detected | System will automatically rewind |
