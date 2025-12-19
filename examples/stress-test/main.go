package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/84hero/evm-scanner/pkg/rpc"
)

// Stress test configuration
type StressConfig struct {
	Duration     time.Duration
	Workers      int
	RequestDelay time.Duration
	ShowProgress bool
}

// Test results
type StressResults struct {
	TotalRequests int64
	SuccessCount  int64
	ErrorCount    int64
	Duration      time.Duration
	QPS           float64
	SuccessRate   float64
}

func main() {
	ctx := context.Background()

	// Configure RPC nodes
	nodes := []rpc.NodeConfig{
		{
			URL:           "https://eth.llamarpc.com",
			Priority:      10,
			RateLimit:     25,
			MaxConcurrent: 10,
		},
		{
			URL:           "https://rpc.ankr.com/eth",
			Priority:      8,
			RateLimit:     10,
			MaxConcurrent: 5,
		},
		{
			URL:           "https://1rpc.io/eth",
			Priority:      5,
			RateLimit:     5,
			MaxConcurrent: 3,
		},
	}

	// Create RPC client
	client, err := rpc.NewClient(ctx, nodes)
	if err != nil {
		log.Fatalf("Failed to create RPC client: %v", err)
	}
	defer client.Close()

	fmt.Println("🔥 RPC Stress Test Suite")
	fmt.Println("=" + string(make([]byte, 60)))
	fmt.Println()

	// Test 1: Light load (10 workers, 10 seconds)
	fmt.Println("📊 Test 1: Light Load (10 workers, 10s)")
	runStressTest(ctx, client, StressConfig{
		Duration:     10 * time.Second,
		Workers:      10,
		RequestDelay: 100 * time.Millisecond,
		ShowProgress: true,
	})
	fmt.Println()

	// Test 2: Medium load (50 workers, 10 seconds)
	fmt.Println("📊 Test 2: Medium Load (50 workers, 10s)")
	runStressTest(ctx, client, StressConfig{
		Duration:     10 * time.Second,
		Workers:      50,
		RequestDelay: 50 * time.Millisecond,
		ShowProgress: true,
	})
	fmt.Println()

	// Test 3: Heavy load (100 workers, 10 seconds)
	fmt.Println("📊 Test 3: Heavy Load (100 workers, 10s)")
	runStressTest(ctx, client, StressConfig{
		Duration:     10 * time.Second,
		Workers:      100,
		RequestDelay: 10 * time.Millisecond,
		ShowProgress: true,
	})
	fmt.Println()

	// Test 4: Burst test (200 concurrent requests)
	fmt.Println("📊 Test 4: Burst Test (200 concurrent)")
	runBurstTest(ctx, client, 200)
	fmt.Println()

	// Test 5: Sustained load (20 workers, 30 seconds)
	fmt.Println("📊 Test 5: Sustained Load (20 workers, 30s)")
	runStressTest(ctx, client, StressConfig{
		Duration:     30 * time.Second,
		Workers:      20,
		RequestDelay: 50 * time.Millisecond,
		ShowProgress: true,
	})
	fmt.Println()

	fmt.Println("✨ All stress tests completed!")
}

func runStressTest(ctx context.Context, client *rpc.MultiClient, config StressConfig) {
	var (
		successCount int64
		errorCount   int64
		wg           sync.WaitGroup
	)

	start := time.Now()
	deadline := start.Add(config.Duration)

	// Progress ticker
	var ticker *time.Ticker
	if config.ShowProgress {
		ticker = time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		go func() {
			for range ticker.C {
				if time.Now().After(deadline) {
					return
				}
				elapsed := time.Since(start)
				success := atomic.LoadInt64(&successCount)
				errors := atomic.LoadInt64(&errorCount)
				total := success + errors
				qps := float64(success) / elapsed.Seconds()
				fmt.Printf("  [%2.0fs] Requests: %d | Success: %d | Errors: %d | QPS: %.2f\n",
					elapsed.Seconds(), total, success, errors, qps)
			}
		}()
	}

	// Launch workers
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				_, err := client.BlockNumber(ctx)
				if err == nil {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}

				if config.RequestDelay > 0 {
					time.Sleep(config.RequestDelay)
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	// Print results
	printResults(StressResults{
		TotalRequests: successCount + errorCount,
		SuccessCount:  successCount,
		ErrorCount:    errorCount,
		Duration:      duration,
		QPS:           float64(successCount) / duration.Seconds(),
		SuccessRate:   float64(successCount) / float64(successCount+errorCount) * 100,
	})
}

func runBurstTest(ctx context.Context, client *rpc.MultiClient, count int) {
	var (
		successCount int64
		errorCount   int64
		wg           sync.WaitGroup
	)

	start := time.Now()

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.BlockNumber(ctx)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	printResults(StressResults{
		TotalRequests: int64(count),
		SuccessCount:  successCount,
		ErrorCount:    errorCount,
		Duration:      duration,
		QPS:           float64(successCount) / duration.Seconds(),
		SuccessRate:   float64(successCount) / float64(count) * 100,
	})
}

func printResults(results StressResults) {
	fmt.Println()
	fmt.Println("Results:")
	fmt.Printf("  Total Requests:  %d\n", results.TotalRequests)
	fmt.Printf("  ✅ Success:      %d (%.2f%%)\n", results.SuccessCount, results.SuccessRate)
	fmt.Printf("  ❌ Errors:       %d (%.2f%%)\n", results.ErrorCount, 100-results.SuccessRate)
	fmt.Printf("  ⏱️  Duration:     %v\n", results.Duration)
	fmt.Printf("  📈 Throughput:   %.2f req/s\n", results.QPS)
	fmt.Printf("  ⚡ Avg Latency:  %.2f ms\n", float64(results.Duration.Milliseconds())/float64(results.TotalRequests))
}
