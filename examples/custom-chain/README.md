# Custom Chain Preset Example

Not all chains are equal. Some have 12-second block times (Ethereum), while others have 1-second or even sub-second block times. Similarly, reorg safety windows vary significantly.

## Features Shown

- **Chain Registration**: How to use `pkg/chain` to register a new network's parameters.
- **Preset Inheritance**: Using the registered preset to configure the `scanner.Config` automatically.
- **Parameter Optimization**: Adjusting `BatchSize` and `Interval` for high-throughput networks.

## Why use Presets?

In a multi-scanner environment (e.g., an indexing service supporting 20+ chains), presets allow you to centralize the network knowledge (block times, safety limits) rather than hardcoding values for every individual scanner instance.

## How to Run

```bash
go run main.go
```
