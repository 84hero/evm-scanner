# SDK Usage Example

This directory contains a full example of how to use `evm-scanner` as a Go library (SDK) in your own applications.

## How to Run

1.  **Prepare configuration**:
    Ensure you have a `config.yaml` in the root directory (you can copy `config.yaml.example`).

2.  **Run the example**:
    ```bash
    go run main.go
    ```

## Key Concepts

- **Chain Presets**: Register custom chain parameters (block time, reorg safety).
- **Client**: High-level RPC client with failover support.
- **Persistence**: Choose between Memory, Redis, or Postgres for tracking progress.
- **Scanner**: The core engine that orchestrates fetching logs and managing cursors.
- **Handler**: Define your own logic to process decoded logs.

## Code Overview

- `pkg/scanner`: Core scanning logic.
- `pkg/rpc`: RPC client wrapper.
- `pkg/decoder`: ABI-based event decoding.
- `pkg/storage`: Progress persistence implementations.
