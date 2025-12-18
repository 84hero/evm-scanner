package main

import (
	"context"
	"fmt"
	"time"

	"github.com/84hero/evm-scanner/pkg/chain"
	"github.com/84hero/evm-scanner/pkg/rpc"
	"github.com/84hero/evm-scanner/pkg/scanner"
	"github.com/84hero/evm-scanner/pkg/storage"
	"github.com/ethereum/go-ethereum/core/types"
)

func main() {
	// [Scenario] We are launching a new AppChain or L2 called "HeroChain"
	// HeroChain has a fast 1-second block time and needs 50 confirmations for safety.
	
	// 1. Register the New Chain Preset
	chain.Register("herochain", chain.Preset{
		ChainID:   "888",
		BlockTime: 1 * time.Second,
		ReorgSafe: 50,  // Requires 50 blocks to be considered final
		BatchSize: 200, // Supports large batch eth_getLogs
	})

	fmt.Println("Registered custom chain: HeroChain (ID: 888)")

	// 2. Setup Scanner with the Registered Preset
	ctx := context.Background()
	client, _ := rpc.NewClient(ctx, []rpc.NodeConfig{{URL: "https://rpc.herochain.io"}}, 5)
	store := storage.NewMemoryStore("herochain_")

	// Get preset values
	preset, _ := chain.Get("herochain")

	config := scanner.Config{
		ChainID:      "herochain",
		BatchSize:    preset.BatchSize, // Use value from preset
		ReorgSafe:    preset.ReorgSafe, // Use value from preset
		Interval:     preset.BlockTime, // Sync interval matches block time
		UseBloom:     true,             // Enable bloom filter for performance
	}

	filter := scanner.NewFilter() // Scan all logs for demonstration

	s := scanner.New(client, store, config, filter)

	s.SetHandler(func(ctx context.Context, logs []types.Log) error {
		fmt.Printf("⛓️ HeroChain: Scanned %d logs at safety height\n", len(logs))
		return nil
	})

	fmt.Printf("Scanner configured for %s (Safety Window: %d blocks)\n", config.ChainID, config.ReorgSafe)
	
	// s.Start(ctx) // Execution omitted for demo purposes
}
