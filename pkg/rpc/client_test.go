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

	mc, err := NewClientWithNodes(ctx, []*Node{node1, node2}, 100)
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
	// Score = 1000 - 0 - (20 * 50) = 0
	assert.Equal(t, int64(0), n.Score(120))
}

func TestExecute_RetryLimit(t *testing.T) {
	ctx := context.Background()
	mockEth := new(MockEthClient)
	// Fail 3 times. Also allow background sync calls.
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(0), errors.New("fail")).Maybe()

	node := NewNodeWithClient(NodeConfig{URL: "node1", Priority: 10}, mockEth)
	mc, _ := NewClientWithNodes(ctx, []*Node{node}, 100)

	_, err := mc.BlockNumber(ctx)
	assert.Error(t, err)
}

func TestExecute_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockEth := new(MockEthClient)
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	node := NewNodeWithClient(NodeConfig{URL: "node1", Priority: 10}, mockEth)
	mc, _ := NewClientWithNodes(ctx, []*Node{node}, 100)

	// Cancel context immediately
	cancel()
	_, err := mc.BlockNumber(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestProxyMethods(t *testing.T) {
	ctx := context.Background()
	mockEth := new(MockEthClient)

	// Expect background sync calls (immediate one)
	mockEth.On("BlockNumber", mock.Anything).Return(uint64(100), nil).Maybe()

	node := NewNodeWithClient(NodeConfig{URL: "node1", Priority: 10}, mockEth)
	mc, _ := NewClientWithNodes(ctx, []*Node{node}, 100)

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
	_, err := NewClient(context.Background(), []NodeConfig{}, 10)
	assert.Error(t, err)

	_, err = NewClientWithNodes(context.Background(), []*Node{}, 10)
	assert.Error(t, err)
}

func TestNewClient_Unreachable(t *testing.T) {
	ctx := context.Background()
	// Truly invalid URLs that fail parsing or dialing immediately
	configs := []NodeConfig{
		{URL: "invalid-scheme://", Priority: 1},
	}
	_, err := NewClient(ctx, configs, 10)
	assert.Error(t, err)
}

func TestNodeGetters(t *testing.T) {
	n := &Node{config: NodeConfig{URL: "http://test", Priority: 5}}
	assert.Equal(t, "http://test", n.URL())
	assert.Equal(t, 5, n.Priority())
}
