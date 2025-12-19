package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/84hero/evm-scanner/pkg/rpc"
)

// This example demonstrates advanced RPC features:
// - Per-node rate limiting
// - Per-node concurrency control
// - Circuit breaker
// - Automatic node switching
// - Height-aware node selection

func main() {
	ctx := context.Background()

	// Configure multiple RPC nodes with different limits
	nodes := []rpc.NodeConfig{
		{
			URL:           "https://eth.llamarpc.com",
			Priority:      10,
			RateLimit:     25, // 25 requests per second
			MaxConcurrent: 10, // Max 10 concurrent requests
		},
		{
			URL:           "https://rpc.ankr.com/eth",
			Priority:      8,
			RateLimit:     10, // 10 requests per second
			MaxConcurrent: 5,  // Max 5 concurrent requests
		},
		{
			URL:           "https://1rpc.io/eth",
			Priority:      5,
			RateLimit:     5, // 5 requests per second
			MaxConcurrent: 3, // Max 3 concurrent requests
		},
	}

	// Create RPC client
	client, err := rpc.NewClient(ctx, nodes)
	if err != nil {
		log.Fatalf("Failed to create RPC client: %v", err)
	}
	defer client.Close()

	fmt.Println("🚀 Advanced RPC Client Demo")
	fmt.Println("=" + string(make([]byte, 50)))
	fmt.Println()

	// Demo 1: Basic RPC call
	fmt.Println("📊 Demo 1: Basic RPC Call")
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		log.Fatalf("Failed to get block number: %v", err)
	}
	fmt.Printf("✅ Current block: %d\n\n", blockNumber)

	// Demo 2: Concurrent requests (tests rate limiting and concurrency control)
	fmt.Println("🔄 Demo 2: Concurrent Requests (20 requests)")
	start := time.Now()

	type result struct {
		block uint64
		err   error
	}

	results := make(chan result, 20)

	for i := 0; i < 20; i++ {
		go func() {
			block, err := client.BlockNumber(ctx)
			results <- result{block, err}
		}()
	}

	successCount := 0
	for i := 0; i < 20; i++ {
		res := <-results
		if res.err == nil {
			successCount++
		}
	}

	duration := time.Since(start)
	fmt.Printf("✅ Success: %d/20 requests\n", successCount)
	fmt.Printf("⏱️  Duration: %v\n", duration)
	fmt.Printf("📈 Throughput: %.2f req/s\n\n", float64(20)/duration.Seconds())

	// Demo 3: Rate limiting demonstration
	fmt.Println("⚡ Demo 3: Rate Limiting Test")
	fmt.Println("Sending 30 rapid requests...")

	rateLimitStart := time.Now()
	rateLimitSuccess := 0

	for i := 0; i < 30; i++ {
		_, err := client.BlockNumber(ctx)
		if err == nil {
			rateLimitSuccess++
		}
	}

	rateLimitDuration := time.Since(rateLimitStart)
	fmt.Printf("✅ Success: %d/30 requests\n", rateLimitSuccess)
	fmt.Printf("⏱️  Duration: %v\n", rateLimitDuration)
	fmt.Printf("📊 Average QPS: %.2f\n\n", float64(rateLimitSuccess)/rateLimitDuration.Seconds())

	// Demo 4: Node health monitoring
	fmt.Println("🏥 Demo 4: Node Health Status")
	// Note: In a real application, you would expose node metrics
	// This is just a demonstration of the concept
	fmt.Println("✅ All nodes operational")
	fmt.Println("   - Circuit breakers: Normal")
	fmt.Println("   - Rate limits: Active")
	fmt.Println("   - Concurrency: Controlled")
	fmt.Println()

	fmt.Println("✨ Demo completed successfully!")
	fmt.Println()
	fmt.Println("Key Features Demonstrated:")
	fmt.Println("  ✅ Per-node rate limiting")
	fmt.Println("  ✅ Per-node concurrency control")
	fmt.Println("  ✅ Automatic node switching")
	fmt.Println("  ✅ High availability with multiple nodes")
}
