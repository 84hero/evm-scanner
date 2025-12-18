package rpc

import (
	"context"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// NodeConfig represents configuration for a single RPC node
type NodeConfig struct {
	URL      string
	Priority int // Initial weight (1-100), higher is more preferred
}

// Node wraps the underlying ethclient and provides health monitoring
type Node struct {
	config NodeConfig
	client EthClient // Interface for underlying ethclient

	// Dynamic metrics (atomic operations)
	errorCount  uint64 // Consecutive error count
	totalErrors uint64 // Total error count
	latency     int64  // Average latency (ms)
	latestBlock uint64 // Latest block height observed by this node
}

// NewNode creates a new RPC node (Production)
func NewNode(ctx context.Context, cfg NodeConfig) (*Node, error) {
	client, err := ethclient.DialContext(ctx, cfg.URL)
	if err != nil {
		return nil, err
	}

	return NewNodeWithClient(cfg, client), nil
}

// NewNodeWithClient initializes Node with a pre-created client (Testing/DI)
func NewNodeWithClient(cfg NodeConfig, client EthClient) *Node {
	return &Node{
		config:  cfg,
		client:  client,
		latency: 0,
	}
}

// URL returns the node address
func (n *Node) URL() string {
	return n.config.URL
}

// Priority returns the configured weight
func (n *Node) Priority() int {
	return n.config.Priority
}

// Score calculates the real-time score of the node. Higher is better.
// Formula: (Priority * 100) - (Latency / 10) - (ConsecutiveErrors * 500)
// Points are also deducted if the node lags too far behind the global max height.
func (n *Node) Score(globalMaxHeight uint64) int64 {
	score := int64(n.config.Priority) * 100

	// Latency penalty (e.g., 200ms latency = -20 points)
	avgLatency := atomic.LoadInt64(&n.latency)
	score -= (avgLatency / 10)

	// Error penalty (consecutive errors are critical)
	errs := atomic.LoadUint64(&n.errorCount)
	score -= int64(errs) * 500

	// Height lag penalty
	myHeight := atomic.LoadUint64(&n.latestBlock)
	if globalMaxHeight > 0 && myHeight < globalMaxHeight {
		lag := globalMaxHeight - myHeight
		if lag > 5 {
			score -= int64(lag) * 50 // -50 points per lagged block
		}
	}

	return score
}

// RecordMetric records result of a call, updating latency and error count
func (n *Node) RecordMetric(start time.Time, err error) {
	duration := time.Since(start).Milliseconds()

	// Simple moving average for latency
	oldLatency := atomic.LoadInt64(&n.latency)
	if oldLatency == 0 {
		atomic.StoreInt64(&n.latency, duration)
	} else {
		// New latency weight 20%
		newLatency := (oldLatency*8 + duration*2) / 10
		atomic.StoreInt64(&n.latency, newLatency)
	}

	if err != nil {
		atomic.AddUint64(&n.errorCount, 1)
		atomic.AddUint64(&n.totalErrors, 1)
	} else {
		// Decrease error count slowly on success to avoid "jitter"
		current := atomic.LoadUint64(&n.errorCount)
		if current > 0 {
			atomic.StoreUint64(&n.errorCount, current-1)
		}
	}
}

// UpdateHeight updates the latest block height for the node
func (n *Node) UpdateHeight(h uint64) {
	current := atomic.LoadUint64(&n.latestBlock)
	if h > current {
		atomic.StoreUint64(&n.latestBlock, h)
	}
}

// GetErrorCount returns the current consecutive error count
func (n *Node) GetErrorCount() uint64 {
	return atomic.LoadUint64(&n.errorCount)
}

// GetTotalErrors returns the total error count
func (n *Node) GetTotalErrors() uint64 {
	return atomic.LoadUint64(&n.totalErrors)
}

// GetLatency returns the average latency in ms
func (n *Node) GetLatency() int64 {
	return atomic.LoadInt64(&n.latency)
}

// GetLatestBlock returns the latest block height observed by this node
func (n *Node) GetLatestBlock() uint64 {
	return atomic.LoadUint64(&n.latestBlock)
}

// Proxy Methods (implement Client interface)

func (n *Node) BlockNumber(ctx context.Context) (uint64, error) {
	start := time.Now()
	h, err := n.client.BlockNumber(ctx)
	n.RecordMetric(start, err)
	if err == nil {
		n.UpdateHeight(h)
	}
	return h, err
}

func (n *Node) ChainID(ctx context.Context) (*big.Int, error) {
	start := time.Now()
	id, err := n.client.ChainID(ctx)
	n.RecordMetric(start, err)
	return id, err
}

func (n *Node) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	start := time.Now()
	h, err := n.client.HeaderByNumber(ctx, number)
	n.RecordMetric(start, err)
	return h, err
}

func (n *Node) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	start := time.Now()
	b, err := n.client.BlockByNumber(ctx, number)
	n.RecordMetric(start, err)
	return b, err
}

func (n *Node) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	start := time.Now()
	logs, err := n.client.FilterLogs(ctx, q)
	n.RecordMetric(start, err)
	return logs, err
}

func (n *Node) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	start := time.Now()
	code, err := n.client.CodeAt(ctx, account, blockNumber)
	n.RecordMetric(start, err)
	return code, err
}

func (n *Node) Close() {
	n.client.Close()
}