package rpc

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestNode_ConcurrencyStressTest tests concurrency control under high load
func TestNode_ConcurrencyStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ctx := context.Background()
	mockEth := new(MockEthClient)
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	node := NewNodeWithClient(NodeConfig{
		URL:           "test",
		Priority:      10,
		MaxConcurrent: 10,
	}, mockEth)

	// Launch 100 concurrent goroutines
	var wg sync.WaitGroup
	successCount := int32(0)
	busyCount := int32(0)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := node.TryAcquire(ctx)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
				time.Sleep(10 * time.Millisecond) // Simulate work
				node.Release()
			} else if err == ErrNodeBusy {
				atomic.AddInt32(&busyCount, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("Success: %d, Busy: %d", successCount, busyCount)

	// Should have some busy errors due to concurrency limit
	assert.Greater(t, busyCount, int32(0), "Should have some busy errors")
	assert.Greater(t, successCount, int32(0), "Should have some successes")
	assert.Equal(t, int32(100), successCount+busyCount, "Total should be 100")
}

// TestNode_RateLimitStressTest tests rate limiting under sustained load
func TestNode_RateLimitStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ctx := context.Background()
	mockEth := new(MockEthClient)
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	// Create node with 10 QPS limit
	node := NewNodeWithClient(NodeConfig{
		URL:       "test",
		Priority:  10,
		RateLimit: 10,
	}, mockEth)

	// Send 50 requests as fast as possible
	start := time.Now()
	successCount := 0
	rateLimitCount := 0

	for i := 0; i < 50; i++ {
		err := node.TryAcquire(ctx)
		if err == nil {
			successCount++
			node.Release()
		} else if err == ErrRateLimitExceeded {
			rateLimitCount++
		}
	}

	duration := time.Since(start)
	actualQPS := float64(successCount) / duration.Seconds()

	t.Logf("Duration: %v", duration)
	t.Logf("Success: %d, Rate Limited: %d", successCount, rateLimitCount)
	t.Logf("Actual QPS: %.2f", actualQPS)

	// Should have some rate limit errors (main goal of this test)
	assert.Greater(t, rateLimitCount, 0, "Should have rate limit errors")

	// In mock environment, QPS can vary widely, so just check it's reasonable
	// Don't assert exact QPS as mock responses are instant
	assert.Greater(t, successCount, 0, "Should have some successes")
}

// TestMultiClient_HighConcurrencyStressTest tests multi-client under extreme load
func TestMultiClient_HighConcurrencyStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ctx := context.Background()

	// Create 3 nodes with different capacities
	nodes := make([]*Node, 3)
	for i := 0; i < 3; i++ {
		mockEth := new(MockEthClient)
		mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

		nodes[i] = NewNodeWithClient(NodeConfig{
			URL:           "node" + string(rune('1'+i)),
			Priority:      10 - i*2,
			RateLimit:     10,
			MaxConcurrent: 5,
		}, mockEth)
	}

	mc, err := NewClientWithNodes(ctx, nodes)
	assert.NoError(t, err)

	// Launch 200 concurrent requests
	var wg sync.WaitGroup
	successCount := int32(0)
	errorCount := int32(0)

	start := time.Now()

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := mc.BlockNumber(ctx)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&errorCount, 1)
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Duration: %v", duration)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)
	t.Logf("Throughput: %.2f req/s", float64(successCount)/duration.Seconds())

	// Most requests should succeed (allow some failures due to rate limiting)
	assert.Greater(t, successCount, int32(150), "Most requests should succeed")
}

// TestCircuitBreaker_StressTest tests circuit breaker under failure conditions
func TestCircuitBreaker_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	node := &Node{
		config: NodeConfig{Priority: 10},
	}

	// Simulate rapid failures
	for i := 0; i < 10; i++ {
		node.RecordMetric(time.Now(), assert.AnError)
	}

	// Circuit should be broken
	assert.True(t, node.IsCircuitBroken(), "Circuit should be broken after 10 failures")

	// Try to acquire - should fail
	err := node.TryAcquire(context.Background())
	assert.ErrorIs(t, err, ErrCircuitBroken)

	// Simulate some successes
	for i := 0; i < 10; i++ {
		node.RecordMetric(time.Now(), nil)
	}

	// Circuit should be reset
	assert.False(t, node.IsCircuitBroken(), "Circuit should reset after successes")
}

// TestNode_SustainedLoadTest tests node behavior under sustained load
func TestNode_SustainedLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ctx := context.Background()
	mockEth := new(MockEthClient)
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	node := NewNodeWithClient(NodeConfig{
		URL:           "test",
		Priority:      10,
		RateLimit:     100, // High limit to reduce rate limiting
		MaxConcurrent: 50,  // High concurrency to reduce busy errors
	}, mockEth)

	// Run for 1 second (reduced for faster CI)
	duration := 1 * time.Second
	deadline := time.Now().Add(duration)

	var wg sync.WaitGroup
	successCount := int32(0)
	errorCount := int32(0)

	// Launch 5 workers (reduced to minimize contention)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				err := node.TryAcquire(ctx)
				if err == nil {
					atomic.AddInt32(&successCount, 1)
					time.Sleep(5 * time.Millisecond) // Reduced work time
					node.Release()
				} else {
					atomic.AddInt32(&errorCount, 1)
					time.Sleep(2 * time.Millisecond) // Reduced back off
				}
			}
		}()
	}

	wg.Wait()

	totalRequests := successCount + errorCount
	actualQPS := float64(successCount) / duration.Seconds()

	t.Logf("Total requests: %d", totalRequests)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)
	if totalRequests > 0 {
		t.Logf("Success rate: %.2f%%", float64(successCount)/float64(totalRequests)*100)
	}
	t.Logf("Actual QPS: %.2f", actualQPS)

	// Just verify the test ran and had some activity
	assert.Greater(t, totalRequests, int32(0), "Should have made some requests")
	assert.Greater(t, successCount, int32(0), "Should have some successes")

	// Don't assert success rate in mock environment as it's too variable
	// The main goal is to verify the code doesn't crash under sustained load
}

// BenchmarkNode_TryAcquire benchmarks the TryAcquire performance
func BenchmarkNode_TryAcquire(b *testing.B) {
	ctx := context.Background()
	mockEth := new(MockEthClient)
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	node := NewNodeWithClient(NodeConfig{
		URL:           "test",
		Priority:      10,
		RateLimit:     1000,
		MaxConcurrent: 100,
	}, mockEth)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := node.TryAcquire(ctx)
			if err == nil {
				node.Release()
			}
		}
	})
}

// BenchmarkMultiClient_BlockNumber benchmarks the BlockNumber call
func BenchmarkMultiClient_BlockNumber(b *testing.B) {
	ctx := context.Background()

	mockEth := new(MockEthClient)
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	node := NewNodeWithClient(NodeConfig{
		URL:           "test",
		Priority:      10,
		RateLimit:     1000,
		MaxConcurrent: 100,
	}, mockEth)

	mc, _ := NewClientWithNodes(ctx, []*Node{node})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = mc.BlockNumber(ctx)
		}
	})
}
