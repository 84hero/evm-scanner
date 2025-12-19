package rpc

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/time/rate"
)

// NodeConfig represents configuration for a single RPC node
type NodeConfig struct {
	URL           string
	Priority      int // Initial weight (1-100), higher is more preferred
	RateLimit     int // QPS limit for this node, 0 means unlimited
	MaxConcurrent int // Max concurrent requests for this node, 0 means unlimited
}

// Node wraps the underlying ethclient and provides health monitoring and metric tracking.
type Node struct {
	config NodeConfig
	client EthClient // Interface for underlying ethclient

	mu          sync.RWMutex
	errorCount  uint64 // Consecutive error count
	totalErrors uint64 // Total error count
	latency     int64  // Average latency (ms)
	latestBlock uint64 // Latest block height observed by this node

	// Concurrency control
	limiter   *rate.Limiter // QPS rate limiter
	semaphore chan struct{} // Concurrency limiter

	// Circuit breaker
	circuitBroken bool
	lastErrorTime time.Time
	breakerMu     sync.RWMutex
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
	node := &Node{
		config:  cfg,
		client:  client,
		latency: 0,
	}

	// Initialize QPS limiter
	if cfg.RateLimit > 0 {
		node.limiter = rate.NewLimiter(rate.Limit(cfg.RateLimit), cfg.RateLimit)
	}

	// Initialize concurrency limiter
	if cfg.MaxConcurrent > 0 {
		node.semaphore = make(chan struct{}, cfg.MaxConcurrent)
	}

	return node
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
	n.mu.RLock()
	defer n.mu.RUnlock()

	score := int64(n.config.Priority) * 100

	// Latency penalty (e.g., 200ms latency = -20 points)
	score -= (n.latency / 10)

	// Error penalty (consecutive errors are critical)
	score -= int64(n.errorCount) * 500

	// Height lag penalty (enhanced)
	if globalMaxHeight > 0 && n.latestBlock < globalMaxHeight {
		lag := globalMaxHeight - n.latestBlock

		if lag > 100 {
			// Severely lagging (>100 blocks) - return extremely low score
			return -10000
		} else if lag > 20 {
			// Significantly lagging (20-100 blocks) - heavy penalty
			score -= int64(lag) * 200
		} else if lag > 5 {
			// Moderately lagging (5-20 blocks) - medium penalty
			score -= int64(lag) * 100
		} else if lag > 0 {
			// Slightly lagging (1-5 blocks) - light penalty
			score -= int64(lag) * 20
		}
	}

	return score
}

// RecordMetric records result of a call, updating latency and error count
func (n *Node) RecordMetric(start time.Time, err error) {
	duration := time.Since(start).Milliseconds()

	n.mu.Lock()
	defer n.mu.Unlock()

	// Simple moving average for latency
	if n.latency == 0 {
		n.latency = duration
	} else {
		// New latency weight 20%
		n.latency = (n.latency*8 + duration*2) / 10
	}

	if err != nil {
		n.errorCount++
		n.totalErrors++
		n.lastErrorTime = time.Now()

		// Trigger circuit breaker if consecutive errors >= 5
		if n.errorCount >= 5 {
			n.TripCircuitBreaker()
		}
	} else {
		// Decrease error count slowly on success to avoid "jitter"
		if n.errorCount > 0 {
			n.errorCount--
		}

		// Reset circuit breaker on success
		if n.errorCount == 0 {
			n.ResetCircuitBreaker()
		}
	}
}

// UpdateHeight updates the latest block height for the node
func (n *Node) UpdateHeight(h uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if h > n.latestBlock {
		n.latestBlock = h
	}
}

// GetErrorCount returns the current consecutive error count
func (n *Node) GetErrorCount() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.errorCount
}

// GetTotalErrors returns the total error count
func (n *Node) GetTotalErrors() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.totalErrors
}

// GetLatency returns the average latency in ms
func (n *Node) GetLatency() int64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.latency
}

// GetLatestBlock returns the latest block height observed by this node
func (n *Node) GetLatestBlock() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.latestBlock
}

// BlockNumber retrieves the latest block height from the node
func (n *Node) BlockNumber(ctx context.Context) (uint64, error) {
	start := time.Now()
	h, err := n.client.BlockNumber(ctx)
	n.RecordMetric(start, err)
	if err == nil {
		n.UpdateHeight(h)
	}
	return h, err
}

// ChainID retrieves the chain ID from the node
func (n *Node) ChainID(ctx context.Context) (*big.Int, error) {
	start := time.Now()
	id, err := n.client.ChainID(ctx)
	n.RecordMetric(start, err)
	return id, err
}

// HeaderByNumber retrieves a block header from the node
func (n *Node) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	start := time.Now()
	h, err := n.client.HeaderByNumber(ctx, number)
	n.RecordMetric(start, err)
	return h, err
}

// BlockByNumber retrieves a full block from the node
func (n *Node) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	start := time.Now()
	b, err := n.client.BlockByNumber(ctx, number)
	n.RecordMetric(start, err)
	return b, err
}

// FilterLogs retrieves logs from the node based on the query
func (n *Node) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	start := time.Now()
	logs, err := n.client.FilterLogs(ctx, q)
	n.RecordMetric(start, err)
	return logs, err
}

// CodeAt retrieves the contract code at a given address
func (n *Node) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	start := time.Now()
	code, err := n.client.CodeAt(ctx, account, blockNumber)
	n.RecordMetric(start, err)
	return code, err
}

// Close closes the underlying RPC connection
func (n *Node) Close() {
	n.client.Close()
}

// Error definitions
var (
	ErrCircuitBroken     = errors.New("node circuit breaker is open")
	ErrRateLimitExceeded = errors.New("node rate limit exceeded")
	ErrNodeBusy          = errors.New("node is busy (max concurrent reached)")
)

// TryAcquire attempts to acquire the node for use (non-blocking)
func (n *Node) TryAcquire(ctx context.Context) error {
	// 1. Check circuit breaker
	if n.IsCircuitBroken() {
		return ErrCircuitBroken
	}

	// 2. QPS rate limiting (non-blocking check)
	if n.limiter != nil {
		if !n.limiter.Allow() {
			return ErrRateLimitExceeded
		}
	}

	// 3. Concurrency control (non-blocking)
	if n.semaphore != nil {
		select {
		case n.semaphore <- struct{}{}:
			// Acquired successfully
		case <-ctx.Done():
			return ctx.Err()
		default:
			return ErrNodeBusy
		}
	}

	return nil
}

// Release releases the node after use
func (n *Node) Release() {
	if n.semaphore != nil {
		<-n.semaphore
	}
}

// CurrentConcurrency returns the current number of concurrent requests
func (n *Node) CurrentConcurrency() int {
	if n.semaphore == nil {
		return 0
	}
	return len(n.semaphore)
}

// IsCircuitBroken checks if the circuit breaker is open
func (n *Node) IsCircuitBroken() bool {
	n.breakerMu.RLock()
	defer n.breakerMu.RUnlock()

	if !n.circuitBroken {
		return false
	}

	// Check if breaker should be reset (30 seconds timeout)
	if time.Since(n.lastErrorTime) > 30*time.Second {
		return false
	}

	return true
}

// TripCircuitBreaker opens the circuit breaker
func (n *Node) TripCircuitBreaker() {
	n.breakerMu.Lock()
	defer n.breakerMu.Unlock()

	n.circuitBroken = true
	n.lastErrorTime = time.Now()
}

// ResetCircuitBreaker closes the circuit breaker
func (n *Node) ResetCircuitBreaker() {
	n.breakerMu.Lock()
	defer n.breakerMu.Unlock()

	n.circuitBroken = false
}

// MeetsHeightRequirement checks if the node has synced to the required height
func (n *Node) MeetsHeightRequirement(requiredHeight uint64) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.latestBlock >= requiredHeight
}
