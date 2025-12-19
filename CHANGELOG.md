# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Per-node QPS rate limiting with configurable `rate_limit` parameter
- Per-node concurrent request control with configurable `max_concurrent` parameter
- Circuit breaker mechanism that automatically trips after 5 consecutive failures
- Automatic circuit breaker recovery after 30-second timeout
- Automatic node switching when nodes are busy, rate-limited, or circuit-broken
- Height requirement checking for node selection to prevent using lagging nodes
- Enhanced node scoring algorithm with stricter height lag penalties
- Comprehensive test coverage for all new features (83.2% code coverage)
- `TryAcquire()` and `Release()` methods for non-blocking node acquisition
- `IsCircuitBroken()` method to check circuit breaker status
- `MeetsHeightRequirement()` method to verify node height
- `pickAvailableNodeWithHeight()` for height-aware node selection

### Changed
- **BREAKING**: Removed `limit` parameter from `rpc.NewClient()` - each node now has independent rate limiting
- **BREAKING**: Removed `limit` parameter from `rpc.NewClientWithNodes()`
- Node scoring algorithm now applies progressive penalties for height lag:
  - Lag > 100 blocks: -10000 points (effectively disabled)
  - Lag 20-100 blocks: -200 points per block
  - Lag 5-20 blocks: -100 points per block
  - Lag 1-5 blocks: -20 points per block
- `execute()` method now uses smart node selection with automatic failover
- Node selection now considers circuit breaker status, rate limits, and concurrency

### Removed
- **BREAKING**: Global rate limiter (previously 20 QPS across all nodes)
- Hard-coded rate limit values from examples and CLI

### Fixed
- Potential issue where all requests could overwhelm a single high-priority node
- Race conditions in concurrent node access
- Inefficient node selection when nodes have different performance characteristics

### Security
- Added protection against node overload through per-node concurrency limits
- Improved resilience with circuit breaker pattern

## [0.1.0] - 2025-12-15

### Added
- Initial release of EVM Scanner
- Multi-node RPC client with automatic failover
- Dynamic node scoring based on priority, latency, and error count
- Block scanning with configurable batch size and interval
- Event log filtering and decoding
- Multiple output sinks (Webhook, PostgreSQL, Message Queue)
- Persistent checkpoint management
- Bloom filter optimization for efficient log filtering
- Comprehensive configuration via YAML files
- CLI tool for quick deployment

[Unreleased]: https://github.com/84hero/evm-scanner/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/84hero/evm-scanner/releases/tag/v0.1.0
