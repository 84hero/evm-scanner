# Webhook Receiver Example

A simple HTTP server to receive and print events sent by the `evm-scanner` webhook output.

## How to Run

1.  **Start the receiver**:
    ```bash
    go run main.go
    ```
    The server will listen on `http://localhost:8080/webhook`.

2.  **Configure the Scanner**:
    In your `app.yaml`, enable the webhook output:
    ```yaml
    outputs:
      webhook:
        enabled: true
        url: "http://localhost:8080/webhook"
        # ... other settings
    ```

3.  **Run the Scanner**:
    ```bash
    ./bin/scanner-cli
    ```

## Expected Output

When the scanner finds matching events, you will see them printed in the receiver's console:
```text
Received 5 events via webhook:
 - [0xdAC17F958D2ee523a2206206994597C13D831ec7] Tx: 0x... | Event: Transfer
```
