package rpc

import (
	"context"
	"errors"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Error definitions
var (
	ErrNoAvailableNodes  = errors.New("no available rpc nodes")
	ErrNoNodeMeetsHeight = errors.New("no node meets the required block height")
)

// MultiClient manages multiple RPC nodes, providing load balancing and failover
type MultiClient struct {
	nodes        []*Node
	globalHeight uint64

	mu sync.RWMutex
}

// NewClient initializes a multi-node client
func NewClient(ctx context.Context, configs []NodeConfig) (*MultiClient, error) {
	if len(configs) == 0 {
		return nil, errors.New("no rpc configs provided")
	}

	nodes := make([]*Node, 0, len(configs))
	for _, cfg := range configs {
		n, err := NewNode(ctx, cfg)
		if err != nil {
			// During initialization, if a node is unreachable, we log it but don't fail,
			// as long as at least one node is connected.
			continue
		}
		nodes = append(nodes, n)
	}

	return NewClientWithNodes(ctx, nodes)
}

// NewClientWithNodes initializes MultiClient with existing nodes (for testing or advanced usage)
func NewClientWithNodes(ctx context.Context, nodes []*Node) (*MultiClient, error) {
	if len(nodes) == 0 {
		return nil, errors.New("failed to connect to any rpc node")
	}

	mc := &MultiClient{
		nodes: nodes,
	}

	// Start background sync task to update node heights and status every 5 seconds
	go mc.startBackgroundSync(ctx)

	return mc, nil
}

// startBackgroundSync periodically polls all nodes to update their heights and scores
func (mc *MultiClient) startBackgroundSync(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Initial sync
	mc.syncNodes(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.syncNodes(ctx)
		}
	}
}

func (mc *MultiClient) syncNodes(ctx context.Context) {
	var maxH uint64
	var wg sync.WaitGroup

	for _, n := range mc.nodes {
		wg.Add(1)
		go func(node *Node) {
			defer wg.Done()
			// Don't use rate limiter for maintenance traffic
			h, err := node.BlockNumber(ctx)
			if err == nil {
				if h > atomic.LoadUint64(&maxH) {
					atomic.StoreUint64(&maxH, h)
				}
			}
		}(n)
	}
	wg.Wait()

	if maxH > 0 {
		atomic.StoreUint64(&mc.globalHeight, maxH)
	}
}

// execute performs an RPC request with retry logic and auto node switching
func (mc *MultiClient) execute(ctx context.Context, op func(*Node) error) error {
	// Max attempts = number of nodes (capped at 3 to avoid long loops)
	attempts := len(mc.nodes)
	if attempts > 3 {
		attempts = 3
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		// Pick an available node (with auto-switching)
		node, err := mc.pickAvailableNode(ctx)
		if err != nil {
			return err
		}

		// Release node after use
		defer node.Release()

		err = op(node)
		if err == nil {
			return nil
		}

		lastErr = err
		// If context is canceled, don't retry
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		// If failed, the node score will automatically decrease via RecordMetric
		// pickAvailableNode might select a different node in next attempt
	}

	return lastErr
}

// ChainID retrieves the chain ID from the best available node
func (mc *MultiClient) ChainID(ctx context.Context) (*big.Int, error) {
	var res *big.Int
	err := mc.execute(ctx, func(n *Node) error {
		var e error
		res, e = n.ChainID(ctx)
		return e
	})
	return res, err
}

// BlockNumber retrieves the latest block height across all nodes (cached if possible)
func (mc *MultiClient) BlockNumber(ctx context.Context) (uint64, error) {
	// Prefer cached global highest height
	h := atomic.LoadUint64(&mc.globalHeight)
	if h > 0 {
		return h, nil
	}
	// If cache empty (at startup), force request
	var res uint64
	err := mc.execute(ctx, func(n *Node) error {
		var e error
		res, e = n.BlockNumber(ctx)
		return e
	})
	return res, err
}

// HeaderByNumber retrieves a block header from the best available node
func (mc *MultiClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var res *types.Header
	err := mc.execute(ctx, func(n *Node) error {
		var e error
		res, e = n.HeaderByNumber(ctx, number)
		return e
	})
	return res, err
}

// BlockByNumber retrieves a full block from the best available node
func (mc *MultiClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	var res *types.Block
	err := mc.execute(ctx, func(n *Node) error {
		var e error
		res, e = n.BlockByNumber(ctx, number)
		return e
	})
	return res, err
}

// FilterLogs retrieves logs from the best available node based on the query
func (mc *MultiClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	var res []types.Log
	err := mc.execute(ctx, func(n *Node) error {
		var e error
		res, e = n.FilterLogs(ctx, q)
		return e
	})
	return res, err
}

// CodeAt retrieves the contract code at a given address from the best available node
func (mc *MultiClient) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	var res []byte
	err := mc.execute(ctx, func(n *Node) error {
		var e error
		res, e = n.CodeAt(ctx, account, blockNumber)
		return e
	})
	return res, err
}

// Close closes all underlying RPC connections
func (mc *MultiClient) Close() {
	for _, n := range mc.nodes {
		n.Close()
	}
}

// pickAvailableNode selects an available node with auto-switching
func (mc *MultiClient) pickAvailableNode(ctx context.Context) (*Node, error) {
	return mc.pickAvailableNodeWithHeight(ctx, 0)
}

// pickAvailableNodeWithHeight selects a node that meets the height requirement
func (mc *MultiClient) pickAvailableNodeWithHeight(ctx context.Context, requiredHeight uint64) (*Node, error) {
	mc.mu.RLock()
	globalH := atomic.LoadUint64(&mc.globalHeight)

	// Create a copy of candidates for sorting
	candidates := make([]*Node, len(mc.nodes))
	copy(candidates, mc.nodes)
	mc.mu.RUnlock()

	if len(candidates) == 0 {
		return nil, ErrNoAvailableNodes
	}

	// Sort by score in descending order
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score(globalH) > candidates[j].Score(globalH)
	})

	// Try to acquire an available node (with auto-switching)
	for _, node := range candidates {
		// 1. Check height requirement
		if requiredHeight > 0 && !node.MeetsHeightRequirement(requiredHeight) {
			continue // Skip nodes that don't meet height requirement
		}

		// 2. Try to acquire the node (non-blocking)
		err := node.TryAcquire(ctx)
		if err == nil {
			return node, nil // Found an available node
		}
		// If node is busy/rate-limited/circuit-broken, try next node
	}

	// All nodes are unavailable, block and wait for the best node
	bestNode := candidates[0]

	// If best node is circuit-broken, return error
	if bestNode.IsCircuitBroken() {
		return nil, ErrNoAvailableNodes
	}

	// If height requirement not met, return error
	if requiredHeight > 0 && !bestNode.MeetsHeightRequirement(requiredHeight) {
		return nil, ErrNoNodeMeetsHeight
	}

	// Otherwise, block and wait for the best node
	return mc.waitForNode(ctx, bestNode)
}

// waitForNode blocks until the node becomes available
func (mc *MultiClient) waitForNode(ctx context.Context, node *Node) (*Node, error) {
	// QPS rate limiting (blocking wait)
	if node.limiter != nil {
		if err := node.limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Concurrency control (blocking wait)
	if node.semaphore != nil {
		select {
		case node.semaphore <- struct{}{}:
			return node, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return node, nil
}
