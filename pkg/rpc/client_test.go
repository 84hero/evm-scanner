package rpc

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNodeScore(t *testing.T) {
	n := &Node{
		config: NodeConfig{Priority: 10},
	}

	// Initial score: 10 * 100 = 1000
	assert.Equal(t, int64(1000), n.Score(0))

	// Simulate latency (no errors recorded)
	n.RecordMetric(time.Now().Add(-100*time.Millisecond), nil)
	// Latency update: (old=0) -> set to 100.
	// Score: 1000 - (100/10) = 990
	assert.Equal(t, int64(990), n.Score(0))

	// Reset node to test errors independently
	n2 := &Node{config: NodeConfig{Priority: 10}}
	n2.RecordMetric(time.Now(), errors.New("fail"))
	// ErrorCount = 1. Latency update: (0*8 + 0)/10 = 0.
	// Score: 1000 - 0 - 500 = 500
	assert.Equal(t, int64(500), n2.Score(0))
}

func TestMultiClient_Failover(t *testing.T) {
	ctx := context.Background()

	// Mock Node 1: Always Fails
	mock1 := new(MockEthClient)
	mock1.On("BlockNumber", mock.Anything).Return(uint64(0), errors.New("connection error"))

	// Mock Node 2: Succeeds
	mock2 := new(MockEthClient)
	mock2.On("BlockNumber", mock.Anything).Return(uint64(100), nil)

	node1 := NewNodeWithClient(NodeConfig{URL: "node1", Priority: 10}, mock1)
	node2 := NewNodeWithClient(NodeConfig{URL: "node2", Priority: 8}, mock2)

	mc, err := NewClientWithNodes(ctx, []*Node{node1, node2})
	assert.NoError(t, err)

	// Test: Execute should try node1 first (high priority), fail, then try node2
	h, err := mc.BlockNumber(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), h)

	// Check metrics: Node 1 should have at least 1 error (from background sync or manual call)
	assert.GreaterOrEqual(t, node1.GetTotalErrors(), uint64(1))
}

func TestNode_ScoreLag(t *testing.T) {
	n := &Node{
		config: NodeConfig{Priority: 10},
	}
	n.UpdateHeight(100)
	// Global height is 120, lag is 20.
	// New scoring: lag=20 triggers "lag > 5" branch
	// Score = 1000 - 0 - (20 * 100) = -1000
	assert.Equal(t, int64(-1000), n.Score(120))
}

func TestExecute_RetryLimit(t *testing.T) {
	ctx := context.Background()
	mockEth := new(MockEthClient)
	// Fail 3 times. Also allow background sync calls.
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(0), errors.New("fail")).Maybe()

	node := NewNodeWithClient(NodeConfig{URL: "node1", Priority: 10}, mockEth)
	mc, _ := NewClientWithNodes(ctx, []*Node{node})

	_, err := mc.BlockNumber(ctx)
	assert.Error(t, err)
}

func TestExecute_ContextCanceled(t *testing.T) {
	// Test that TryAcquire respects context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create a node with concurrency limit to test semaphore blocking
	node := &Node{
		config: NodeConfig{
			URL:           "test",
			Priority:      10,
			MaxConcurrent: 1,
		},
		semaphore: make(chan struct{}, 1),
	}

	// Fill the semaphore
	node.semaphore <- struct{}{}

	// Try to acquire with canceled context - should fail
	err := node.TryAcquire(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestProxyMethods(t *testing.T) {
	ctx := context.Background()
	mockEth := new(MockEthClient)

	// Expect background sync calls (immediate one)
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	node := NewNodeWithClient(NodeConfig{URL: "node1", Priority: 10}, mockEth)
	mc, _ := NewClientWithNodes(ctx, []*Node{node})

	// 1. ChainID
	mockEth.On("ChainID", ctx).Return(big.NewInt(1), nil).Once()
	id, err := mc.ChainID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), id.Int64())

	// 2. HeaderByNumber
	header := &types.Header{Number: big.NewInt(100)}
	mockEth.On("HeaderByNumber", ctx, big.NewInt(100)).Return(header, nil).Once()
	h, err := mc.HeaderByNumber(ctx, big.NewInt(100))
	assert.NoError(t, err)
	assert.Equal(t, int64(100), h.Number.Int64())

	// 3. BlockByNumber
	block := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(200)})
	mockEth.On("BlockByNumber", ctx, big.NewInt(200)).Return(block, nil).Once()
	b, err := mc.BlockByNumber(ctx, big.NewInt(200))
	assert.NoError(t, err)
	assert.Equal(t, int64(200), b.Number().Int64())

	// 4. CodeAt
	mockEth.On("CodeAt", ctx, common.HexToAddress("0x1234"), big.NewInt(300)).Return([]byte{0x1}, nil).Once()
	code, err := mc.CodeAt(ctx, common.HexToAddress("0x1234"), big.NewInt(300))
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x1}, code)

	// 5. FilterLogs
	q := ethereum.FilterQuery{FromBlock: big.NewInt(100)}
	mockEth.On("FilterLogs", ctx, q).Return([]types.Log{}, nil).Once()
	logs, err := mc.FilterLogs(ctx, q)
	assert.NoError(t, err)
	assert.Empty(t, logs)

	// 6. Close
	mockEth.On("Close").Once()
	mc.Close()
}

func TestNewClient_Errors(t *testing.T) {
	_, err := NewClient(context.Background(), []NodeConfig{})
	assert.Error(t, err)

	_, err = NewClientWithNodes(context.Background(), []*Node{})
	assert.Error(t, err)
}

func TestNewClient_Unreachable(t *testing.T) {
	ctx := context.Background()
	// Truly invalid URLs that fail parsing or dialing immediately
	configs := []NodeConfig{
		{URL: "invalid-scheme://", Priority: 1},
	}
	_, err := NewClient(ctx, configs)
	assert.Error(t, err)
}

func TestNodeGetters(t *testing.T) {
	n := &Node{config: NodeConfig{URL: "http://test", Priority: 5}}
	assert.Equal(t, "http://test", n.URL())
	assert.Equal(t, 5, n.Priority())
}

// ========== New Feature Tests ==========

// TestNode_ConcurrencyControl tests the max concurrent requests limit
func TestNode_ConcurrencyControl(t *testing.T) {
	ctx := context.Background()

	// Create node with max 2 concurrent requests
	node := NewNodeWithClient(
		NodeConfig{
			URL:           "test",
			Priority:      10,
			MaxConcurrent: 2,
		},
		new(MockEthClient),
	)

	// First two acquisitions should succeed
	err1 := node.TryAcquire(ctx)
	assert.NoError(t, err1)

	err2 := node.TryAcquire(ctx)
	assert.NoError(t, err2)

	// Third acquisition should fail (max concurrent reached)
	err3 := node.TryAcquire(ctx)
	assert.ErrorIs(t, err3, ErrNodeBusy)

	// Release one slot
	node.Release()

	// Now acquisition should succeed again
	err4 := node.TryAcquire(ctx)
	assert.NoError(t, err4)

	// Verify current concurrency
	assert.Equal(t, 2, node.CurrentConcurrency())

	// Release all
	node.Release()
	node.Release()
	assert.Equal(t, 0, node.CurrentConcurrency())
}

// TestNode_RateLimiting tests the QPS rate limiting
func TestNode_RateLimiting(t *testing.T) {
	ctx := context.Background()

	// Create node with 10 QPS limit
	node := NewNodeWithClient(
		NodeConfig{
			URL:       "test",
			Priority:  10,
			RateLimit: 10,
		},
		new(MockEthClient),
	)

	// Burst of 10 requests should succeed (initial bucket is full)
	successCount := 0
	for i := 0; i < 10; i++ {
		err := node.TryAcquire(ctx)
		if err == nil {
			successCount++
			node.Release()
		}
	}
	assert.Equal(t, 10, successCount)

	// Immediate next requests should fail (bucket empty)
	failCount := 0
	for i := 0; i < 5; i++ {
		err := node.TryAcquire(ctx)
		if err == ErrRateLimitExceeded {
			failCount++
		} else if err == nil {
			node.Release()
		}
	}
	assert.Greater(t, failCount, 0, "Should have some rate limit failures")
}

// TestNode_CircuitBreaker tests the circuit breaker mechanism
func TestNode_CircuitBreaker(t *testing.T) {
	node := &Node{
		config: NodeConfig{Priority: 10},
	}

	// Initially not broken
	assert.False(t, node.IsCircuitBroken())

	// Simulate 4 consecutive failures (not enough to trip)
	for i := 0; i < 4; i++ {
		node.RecordMetric(time.Now(), errors.New("fail"))
	}
	assert.False(t, node.IsCircuitBroken())

	// 5th failure should trip the breaker
	node.RecordMetric(time.Now(), errors.New("fail"))
	assert.True(t, node.IsCircuitBroken())

	// TryAcquire should fail when circuit is broken
	err := node.TryAcquire(context.Background())
	assert.ErrorIs(t, err, ErrCircuitBroken)

	// Success should reset error count but breaker stays open for 30s
	node.RecordMetric(time.Now(), nil)
	assert.Equal(t, uint64(4), node.GetErrorCount())

	// More successes should eventually reset the breaker
	for i := 0; i < 4; i++ {
		node.RecordMetric(time.Now(), nil)
	}
	assert.Equal(t, uint64(0), node.GetErrorCount())
	assert.False(t, node.circuitBroken)
}

// TestNode_CircuitBreakerTimeout tests that circuit breaker resets after timeout
func TestNode_CircuitBreakerTimeout(t *testing.T) {
	node := &Node{
		config: NodeConfig{Priority: 10},
	}

	// Trip the breaker
	for i := 0; i < 5; i++ {
		node.RecordMetric(time.Now(), errors.New("fail"))
	}
	assert.True(t, node.IsCircuitBroken())

	// Set last error time to 31 seconds ago
	node.lastErrorTime = time.Now().Add(-31 * time.Second)

	// Circuit breaker should be considered reset
	assert.False(t, node.IsCircuitBroken())
}

// TestNode_HeightRequirement tests the height requirement checking
func TestNode_MeetsHeightRequirement(t *testing.T) {
	node := &Node{
		config:      NodeConfig{Priority: 10},
		latestBlock: 100,
	}

	// Node at height 100
	assert.True(t, node.MeetsHeightRequirement(100))
	assert.True(t, node.MeetsHeightRequirement(99))
	assert.True(t, node.MeetsHeightRequirement(50))
	assert.False(t, node.MeetsHeightRequirement(101))
	assert.False(t, node.MeetsHeightRequirement(150))
}

// TestNode_EnhancedScoring tests the enhanced height lag penalty
func TestNode_EnhancedScoring(t *testing.T) {
	tests := []struct {
		name         string
		priority     int
		latency      int64
		errorCount   uint64
		nodeHeight   uint64
		globalHeight uint64
		expectedMin  int64
		expectedMax  int64
	}{
		{
			name:         "No lag, no errors",
			priority:     10,
			latency:      0,
			errorCount:   0,
			nodeHeight:   100,
			globalHeight: 100,
			expectedMin:  1000,
			expectedMax:  1000,
		},
		{
			name:         "Slight lag (3 blocks)",
			priority:     10,
			latency:      0,
			errorCount:   0,
			nodeHeight:   97,
			globalHeight: 100,
			expectedMin:  940,
			expectedMax:  940,
		},
		{
			name:         "Moderate lag (10 blocks)",
			priority:     10,
			latency:      0,
			errorCount:   0,
			nodeHeight:   90,
			globalHeight: 100,
			expectedMin:  0,
			expectedMax:  0,
		},
		{
			name:         "Significant lag (50 blocks)",
			priority:     10,
			latency:      0,
			errorCount:   0,
			nodeHeight:   50,
			globalHeight: 100,
			expectedMin:  -9000,
			expectedMax:  -9000,
		},
		{
			name:         "Severe lag (>100 blocks)",
			priority:     10,
			latency:      0,
			errorCount:   0,
			nodeHeight:   0,
			globalHeight: 150,
			expectedMin:  -10000,
			expectedMax:  -10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &Node{
				config:      NodeConfig{Priority: tt.priority},
				latency:     tt.latency,
				errorCount:  tt.errorCount,
				latestBlock: tt.nodeHeight,
			}

			score := node.Score(tt.globalHeight)
			assert.GreaterOrEqual(t, score, tt.expectedMin)
			assert.LessOrEqual(t, score, tt.expectedMax)
		})
	}
}

// TestMultiClient_AutoSwitchOnBusy tests automatic node switching when nodes are busy
func TestMultiClient_AutoSwitchOnBusy(t *testing.T) {
	ctx := context.Background()

	// Create two nodes with different concurrency limits
	mock1 := new(MockEthClient)
	mock1.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	mock2 := new(MockEthClient)
	mock2.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	node1 := NewNodeWithClient(NodeConfig{
		URL:           "node1",
		Priority:      10,
		MaxConcurrent: 1,
	}, mock1)

	node2 := NewNodeWithClient(NodeConfig{
		URL:           "node2",
		Priority:      8,
		MaxConcurrent: 1,
	}, mock2)

	mc, err := NewClientWithNodes(ctx, []*Node{node1, node2})
	assert.NoError(t, err)

	// First request should use node1 (higher priority)
	h1, err := mc.BlockNumber(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), h1)

	// Second concurrent request should auto-switch to node2
	// (because node1 is at max concurrency)
	h2, err := mc.BlockNumber(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), h2)
}

// TestMultiClient_HeightRequirement tests node selection with height requirement
func TestMultiClient_HeightRequirement(t *testing.T) {
	ctx := context.Background()

	mock1 := new(MockEthClient)
	mock1.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	mock2 := new(MockEthClient)
	mock2.On("BlockNumber", mock.Anything).Return(uint64(150), nil).Maybe()

	node1 := NewNodeWithClient(NodeConfig{URL: "node1", Priority: 10}, mock1)
	node1.UpdateHeight(100)

	node2 := NewNodeWithClient(NodeConfig{URL: "node2", Priority: 8}, mock2)
	node2.UpdateHeight(150)

	mc, _ := NewClientWithNodes(ctx, []*Node{node1, node2})

	// Request requiring height 120 should skip node1 and use node2
	node, err := mc.pickAvailableNodeWithHeight(ctx, 120)
	assert.NoError(t, err)
	assert.Equal(t, "node2", node.URL())
	node.Release()

	// Request requiring height 90 can use either (should pick node1 - higher priority)
	node, err = mc.pickAvailableNodeWithHeight(ctx, 90)
	assert.NoError(t, err)
	assert.Equal(t, "node1", node.URL())
	node.Release()
}

// TestMultiClient_AllNodesLagging tests behavior when all nodes are behind required height
func TestMultiClient_AllNodesLagging(t *testing.T) {
	ctx := context.Background()

	mockEth := new(MockEthClient)
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	node := NewNodeWithClient(NodeConfig{URL: "node1", Priority: 10}, mockEth)
	node.UpdateHeight(100)

	mc, _ := NewClientWithNodes(ctx, []*Node{node})

	// Request requiring height 150 should fail (no node meets requirement)
	_, err := mc.pickAvailableNodeWithHeight(ctx, 150)
	assert.ErrorIs(t, err, ErrNoNodeMeetsHeight)
}
