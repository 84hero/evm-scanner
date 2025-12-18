# Custom Sink Implementation Example

This example shows how to extend the `evm-scanner` framework by implementing your own `Output` (Sink) interface. This is useful for sending blockchain events to internal tools, proprietary APIs, or unsupported notification services (like Slack, Discord, or Telegram).

## How to Implement a Sink

To create a custom sink, you only need to implement three methods:

```go
type Output interface {
    Name() string
    Send(ctx context.Context, logs []DecodedLog) error
    Close() error
}
```

## How to Run

```bash
go run main.go
```

## Why use this?

The framework's `SetHandler` allows you to write any logic, but implementing the `sink.Output` interface allows you to reuse your sink across different projects or scanners, and integrate it into a `Pipeline` (as shown in the `multi-sink` example).
