# Custom Sinks

EVM Scanner can be used as a library in Go projects, allowing you to extend its functionality with custom output destinations (Sinks).

## Implementation

To create a custom sink, implement the `sink.Output` interface:

```go
type Output interface {
    // Name returns the unique identifier for the sink
    Name() string
    
    // Send processes the batch of logs
    Send(ctx context.Context, logs []DecodedLog) error
    
    // Close releases any allocated resources
    Close() error
}
```

## Example: Slack Notification Sink

```go
package main

import (
    "context"
    "fmt"
    "github.com/84hero/evm-scanner/pkg/sink"
)

type SlackSink struct {
    WebhookURL string
}

func (s *SlackSink) Name() string { return "slack" }

func (s *SlackSink) Send(ctx context.Context, logs []sink.DecodedLog) error {
    for _, l := range logs {
        fmt.Printf("[Slack] New transaction detected: %s\n", l.Log.TxHash.Hex())
    }
    return nil
}

func (s *SlackSink) Close() error { return nil }
```

## Integrating with the Scanner

Initialize the scanner manually and call your custom sink within the handler:

```go
func main() {
    ctx := context.Background()
    
    // Setup RPC and Storage
    client, _ := rpc.NewClient(ctx, nodes, 5)
    store := storage.NewMemoryStore("prefix_")
    
    // Your Custom Sink
    mySink := &SlackSink{WebhookURL: "https://..."}
    
    s := scanner.New(client, store, cfg, filter)
    
    s.SetHandler(func(ctx context.Context, logs []types.Log) error {
        decoded := wrapLogs(logs)
        return mySink.Send(ctx, decoded)
    })
    
    s.Start(ctx)
}
```

## Why use Custom Sinks?

1. **Internal Integration**: Call private microservices or permission systems.
2. **Specific Formatting**: Transform raw data into business-specific alert templates.
3. **Conditional Logic**: Route messages based on content (e.g., high-value vs low-value).
4. **Performance Metrics**: Add custom Prometheus labels for specific events.

## See Also
- Full code example in [examples/custom-sink](../../examples/custom-sink/main.go).
- Default implementations in [pkg/sink](../../pkg/sink/).
