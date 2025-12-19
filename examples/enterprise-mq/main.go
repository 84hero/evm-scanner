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

	// 1. Setup RPC
	client, _ := rpc.NewClient(ctx, []rpc.NodeConfig{{URL: "https://rpc.ankr.com/eth"}})
	store := storage.NewMemoryStore("mq_demo_")

	// 2. Initialize Kafka Sink (Requires a running Kafka broker)
	kafkaBrokers := []string{"localhost:9092"}
	if broker := os.Getenv("KAFKA_BROKER"); broker != "" {
		kafkaBrokers = []string{broker}
	}

	kafkaSink, err := sink.NewKafkaOutput(kafkaBrokers, "evm-events", "", "")
	if err != nil {
		log.Printf("⚠️ Warning: Could not connect to Kafka: %v. Running in log-only mode.", err)
	} else {
		defer kafkaSink.Close()
		fmt.Println("Connected to Kafka at", kafkaBrokers)
	}

	// 3. Define Filter (e.g., Uniswap V2 Pair Created)
	factoryAddr := common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f")
	pairCreatedTopic := crypto.Keccak256Hash([]byte("PairCreated(address,address,address,uint256)"))
	filter := scanner.NewFilter().AddContract(factoryAddr).SetTopic(0, pairCreatedTopic)

	// 4. Scanner Config
	s := scanner.New(client, store, scanner.Config{
		ChainID: "ethereum", Rewind: 50, Interval: 10 * time.Second,
	}, filter)

	s.SetHandler(func(ctx context.Context, logs []types.Log) error {
		decodedLogs := make([]sink.DecodedLog, len(logs))
		for i, l := range logs {
			decodedLogs[i] = sink.DecodedLog{
				Log:       l,
				EventName: "PairCreated",
			}
		}

		// Dispatch to Kafka if available
		if kafkaSink != nil {
			if err := kafkaSink.Send(ctx, decodedLogs); err != nil {
				return fmt.Errorf("kafka send error: %w", err)
			}
			fmt.Printf("✅ Sent %d events to Kafka topic 'evm-events'\n", len(logs))
		} else {
			fmt.Printf("📝 Captured %d events (Kafka offline)\n", len(logs))
		}
		return nil
	})

	fmt.Println("Enterprise MQ Example started...")
	go s.Start(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
