# Multi-Sink Pipeline Example

This example demonstrates how to build a data pipeline that dispatches scanned events to multiple destinations (Sinks) simultaneously.

## Features Shown

- **Multiple Contracts**: Monitoring both USDT and USDC.
- **Persistence**: Using Redis for tracking progress (optional).
- **Pipeline Architecture**: Sending the same event data to:
    - **Console**: For real-time monitoring.
    - **File**: For persistent logging (`events.jsonl`).
- **Graceful Shutdown**: Handling OS signals to stop the scanner safely.

## How to Run

1.  **Run the example**:
    ```bash
    go run main.go
    ```

2.  **Check Output**:
    - Watch the terminal for console JSON output.
    - Check the `events.jsonl` file created in the current directory.

## Configuration Notes

- The example uses Ankr's public RPC node.
- If `REDIS_ADDR` environment variable is set, it will attempt to use Redis for cursor persistence.
