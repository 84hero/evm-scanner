# Architecture Design

This document describes the system architecture, core components, and data flow of EVM Scanner.

## System Overview

EVM Scanner is a high-performance, industrial-grade event scanning and indexing framework for Ethereum and EVM-compatible chains. It uses a producer-consumer model to fetch blockchain data through a multi-node RPC pool and supports various downstream outputs.

## Core Components

### 1. Scanner
The heart of the system responsible for:
- Block height synchronization and progress management.
- Batch fetching of block logs.
- Handling chain reorganizations (reorgs) via rewind logic.
- Bloom Filter optimization for speed.

### 2. RPC Client Pool
Provides high availability and reliability:
- **Failover**: Automatically switches to backup nodes if the primary node fails.
- **Priority Management**: Controls node usage order based on assigned priority.
- **Retry Mechanism**: Built-in exponential backoff to handle network jitters.

### 3. Event Decoder
Parses raw blockchain logs:
- Loads ABI definitions.
- Decodes `Topics` and `Data` into human-readable JSON.
- Supports concurrent parsing for multiple contracts and events.

### 4. Storage Engine
Persists the scanning progress (Cursor):
- **Postgres**: Recommended for production, providing high data consistency.
- **Redis**: Ideal for high-frequency updates and extreme performance.
- **Memory**: Used for testing or one-time scans.

### 5. Sink Manager (Outputs)
Dispatches processed events to various destinations:
- **Webhooks**: HTTP POST with signing secrets and retry logic.
- **Message Queues**: Support for Kafka, RabbitMQ, and Redis.
- **Databases**: Direct writing to Postgres.
- **Console/File**: For debugging and logging.

## Data Flow

1. **Fetch Blocks**: The Scanner polls the RPC node to get the latest block height.
2. **Filter Logs**: Fetches logs based on configured contract addresses and topics using `eth_getLogs`.
3. **Decode**: The Decoder converts raw logs into structured data using ABIs.
4. **Persist Progress**: Scanner updates the current block height in the storage engine.
5. **Dispatch**: Sink Manager concurrently sends data to all enabled output targets.

## Fault Recovery

- **Process Restart**: Automatically resumes from the last saved `Cursor` position.
- **RPC Failure**: Automatically fails over to the next node in the pool.
- **Output Failure**: Webhook and other sinks support retries to ensure delivery during temporary downstream outages.

## Performance Optimization

- **Concurrency**: Parallel log decoding and downstream dispatching.
- **Batching**: Configurable `batch_size` to optimize RPC roundtrips.
- **Node-side Filtering**: Utilizes the EVM Bloom Filter to skip uninteresting blocks efficiently.
