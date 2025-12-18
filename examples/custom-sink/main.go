package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/84hero/evm-scanner/pkg/rpc"
	"github.com/84hero/evm-scanner/pkg/scanner"
	"github.com/84hero/evm-scanner/pkg/sink"
	"github.com/84hero/evm-scanner/pkg/storage"
	"github.com/ethereum/go-ethereum/core/types"
)

// SlackSink is a custom implementation of the sink.Output interface
type SlackSink struct {
	WebhookURL string
}

func (s *SlackSink) Name() string { return "slack" }

func (s *SlackSink) Send(ctx context.Context, logs []sink.DecodedLog) error {
	for _, l := range logs {
		// In a real app, you would send an actual HTTP request to Slack here
		fmt.Printf("[Slack Notification] 📢 New event in tx %s\n", l.Log.TxHash.Hex())
	}
	return nil
}

func (s *SlackSink) Close() error { return nil }

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup
	client, _ := rpc.NewClient(ctx, []rpc.NodeConfig{{URL: "https://rpc.ankr.com/eth"}}, 5)
	store := storage.NewMemoryStore("custom_sink_")

	// 2. Register our Custom Sink
	mySlackSink := &SlackSink{WebhookURL: "https://hooks.slack.com/services/வதற்காக"}

	// 3. Scanner logic
	filter := scanner.NewFilter() // Scan everything for this demo
	s := scanner.New(client, store, scanner.Config{
		ChainID: "ethereum", Rewind: 1, Interval: 5 * time.Second,
	}, filter)

	s.SetHandler(func(ctx context.Context, logs []types.Log) error {
		decoded := make([]sink.DecodedLog, len(logs))
		for i, l := range logs {
			decoded[i] = sink.DecodedLog{Log: l}
		}
		
		// Use our custom sink
		return mySlackSink.Send(ctx, decoded)
	})

	fmt.Println("Scanner running with Custom Slack Sink...")
	go s.Start(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
