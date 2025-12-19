package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/84hero/evm-scanner/pkg/rpc"
	"github.com/84hero/evm-scanner/pkg/scanner"
	"github.com/84hero/evm-scanner/pkg/sink"
	"github.com/84hero/evm-scanner/pkg/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup RPC Client (Using public nodes for demo)
	rpcCfg := []rpc.NodeConfig{
		{URL: "https://rpc.ankr.com/eth", Priority: 1},
	}
	client, err := rpc.NewClient(ctx, rpcCfg, 5)
	if err != nil {
		log.Fatalf("Failed to init RPC client: %v", err)
	}

	// 2. Define Filter (USDT + USDC Transfer events)
	usdtAddr := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	usdcAddr := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	transferTopic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

	filter := scanner.NewFilter().
		AddContract(usdtAddr).
		AddContract(usdcAddr).
		SetTopic(0, transferTopic)

	// 3. Setup Persistence (Redis for distributed cursor management)
	// Fallback to memory if REDIS_ADDR is not set
	var store storage.Persistence
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		store, _ = storage.NewRedisStore(addr, "", 0, "indexer_")
		fmt.Println("Using Redis for persistence")
	} else {
		store = storage.NewMemoryStore("indexer_")
		fmt.Println("Using Memory for persistence")
	}

	// 4. Setup Multiple Sinks (Pipeline)
	var outputs []sink.Output

	// Console Sink
	outputs = append(outputs, sink.NewConsoleOutput())

	// File Sink
	if fo, err := sink.NewFileOutput("events.jsonl"); err == nil {
		outputs = append(outputs, fo)
		fmt.Println("File sink enabled: events.jsonl")
	}

	// 5. Initialize Scanner
	scanCfg := scanner.Config{
		ChainID:   "ethereum",
		Rewind:    10, // Start from 10 blocks ago
		Interval:  5 * time.Second,
		ReorgSafe: 2,
		BatchSize: 10,
	}

	s := scanner.New(client, store, scanCfg, filter)

	// 6. Set Handler to dispatch to all Sinks
	s.SetHandler(func(ctx context.Context, logs []types.Log) error {
		decodedLogs := make([]sink.DecodedLog, len(logs))
		for i, l := range logs {
			decodedLogs[i] = sink.DecodedLog{Log: l}
		}

		fmt.Printf(">>> Processed %d logs\n", len(logs))
		for _, out := range outputs {
			if err := out.Send(ctx, decodedLogs); err != nil {
				log.Printf("Sink %s error: %v", out.Name(), err)
			}
		}
		return nil
	})

	// 7. Start & Handle Signals
	go s.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down...")
}
