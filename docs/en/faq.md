# FAQ

Frequently asked questions and common solutions for EVM Scanner.

## General

### 1. Which chains are supported?
Any chain that satisfies the standard Ethereum JSON-RPC specification, including:
- Ethereum, BSC (Binance Smart Chain), Polygon (Matic)
- Arbitrum, Optimism, Base
- Avalanche (C-Chain), Fantom
- Testnets (Sepolia, Goerli, Mumbai, etc.)

### 2. Why am I not seeing any events?
Check the following:
- **Contract Address**: Ensure it starts with `0x` and matches exactly.
- **Topic0 Hash**: Ensure you are using the Keccak256 hash of the event signature.
- **Start Block**: If the events happened at block 18m and you started at 19m, you won't see them.
- **RPC Sync**: Ensure your RPC node is synchronized with the network.

### 3. How are chain reorgs handled?
Scanner uses the `confirmations` parameter. 
- **Ethereum**: 12-64 confirmations.
- **Polygon/BSC**: 100-200 (reorgs are deeper and more frequent here).
- If a reorg happens within the confirmation window, the scanner rewinds and re-scans automatically.

## Performance

### 4. What is the ideal `batch_size`?
- **Alchemy/Infura**: 50-100.
- **Public Nodes**: 10-20 (to avoid rate limiting).
- **Private Nodes**: 500-1000 for high-speed synchronization.

### 5. Redis vs. PostgreSQL for storage?
- **Redis**: Best for high-performance and low-latency tracking.
- **Postgres**: Recommended for production for better durability and easier backups.

## Sinks

### 6. What happens if a Webhook fails?
If `retry` is enabled, the system follows an exponential backoff. If all attempts fail, the message is dropped from the buffer (unless using a reliable queue like Kafka).

### 7. Can I send data to multiple places?
Yes. Simply enable multiple `outputs` in your `app.yaml`. For example, you can print to your console and write to Postgres simultaneously.

## Development

### 8. Can I run multiple scanners in one process?
Yes. If using the SDK, you can instantiate multiple `scanner.New` objects with different filters and RPC configs. For CLI usage, we recommend separate processes for better resource monitoring.
