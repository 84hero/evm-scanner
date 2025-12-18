# Custom ABI Decoder Example

This example demonstrates how to use the `pkg/decoder` package to convert raw hexadecimal blockchain logs into human-readable Go maps and types.

## Features Shown

- **ABI Loading**: Loading an ERC20 ABI string.
- **Dynamic Decoding**: Using `ABIWrapper` to match and parse log data.
- **Type Casting**: How to cast the decoded interface values back to `common.Address` or other types.

## How to Run

```bash
go run main.go
```

## Implementation Detail

The `decoder` package simplifies the complex logic of parsing topics and data fields according to the Ethereum ABI specification. By providing an ABI, you get access to the field names defined in the smart contract (e.g., `from`, `to`, `value`).
