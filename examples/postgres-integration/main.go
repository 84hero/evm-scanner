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
	// Connection string: postgres://user:password@localhost:5432/dbname?sslmode=disable
	pgURL := os.Getenv("PG_URL")
	if pgURL == "" {
		log.Fatal("PG_URL environment variable is required (e.g., postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable)")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Init RPC
	client, _ := rpc.NewClient(ctx, []rpc.NodeConfig{{URL: "https://rpc.ankr.com/eth"}}, 5)

	// 2. Init Postgres Storage (for progress tracking)
	// This will create a 'cursors' table with prefix 'demo_'
	store, err := storage.NewPostgresStore(pgURL, "demo_")
	if err != nil {
		log.Fatalf("Failed to init storage: %v", err)
	}

	// 3. Init Postgres Sink (for event data)
	// This will create a 'contract_events' table to store logs
	pgSink, err := sink.NewPostgresOutput(pgURL, "contract_events")
	if err != nil {
		log.Fatalf("Failed to init sink: %v", err)
	}
	defer pgSink.Close()

	// 4. Define Filter
	usdtAddr := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	transferTopic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	filter := scanner.NewFilter().AddContract(usdtAddr).SetTopic(0, transferTopic)

	// 5. Run Scanner
	s := scanner.New(client, store, scanner.Config{
		ChainID: "ethereum", Rewind: 10, Interval: 5 * time.Second,
	}, filter)

	s.SetHandler(func(ctx context.Context, logs []types.Log) error {
		decodedLogs := make([]sink.DecodedLog, len(logs))
		for i, l := range logs {
			decodedLogs[i] = sink.DecodedLog{
				Log:       l,
				EventName: "Transfer",
			}
		}
		// Data is saved to the 'contract_events' table
		return pgSink.Send(ctx, decodedLogs)
	})

	fmt.Println("Scanner running with PostgreSQL storage and sink...")
	go s.Start(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
