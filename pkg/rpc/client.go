package rpc

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/time/rate"
)

// ErrNoAvailableNodes is returned when no RPC nodes are currently healthy or reachable.
var ErrNoAvailableNodes = errors.New("no available rpc nodes")

// MultiClient manages multiple RPC nodes, providing load balancing and failover
type MultiClient struct {
	nodes        []*Node
	globalHeight uint64
	limiter      *rate.Limiter

	mu sync.RWMutex
}

// NewClient initializes a multi-node client
// limit: maximum requests per second (RPS)
func NewClient(ctx context.Context, configs []NodeConfig, limit int) (*MultiClient, error) {
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

	return NewClientWithNodes(ctx, nodes, limit)
}

// NewClientWithNodes initializes MultiClient with existing nodes (for testing or advanced usage)
func NewClientWithNodes(ctx context.Context, nodes []*Node, limit int) (*MultiClient, error) {
	if len(nodes) == 0 {
		return nil, errors.New("failed to connect to any rpc node")
	}

	mc := &MultiClient{
		nodes:   nodes,
		limiter: rate.NewLimiter(rate.Limit(limit), limit),
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

// pickBestNode selects the best node based on scores
func (mc *MultiClient) pickBestNode() *Node {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	globalH := atomic.LoadUint64(&mc.globalHeight)

	// Create a copy of candidates for sorting to avoid lock contention
	candidates := make([]*Node, len(mc.nodes))
	copy(candidates, mc.nodes)

	if len(candidates) == 1 {
		return candidates[0]
	}

	// Sort by score in descending order
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score(globalH) > candidates[j].Score(globalH)
	})

	// Simple load balancing: if top two nodes have similar scores, pick one randomly
	// to avoid overloading the first node.
	top1 := candidates[0]
	if len(candidates) > 1 {
		top2 := candidates[1]
		// If score difference is small (e.g., just a slight latency difference), pick top2 with 50% probability
		if (top1.Score(globalH) - top2.Score(globalH)) < 50 {
			if rand.Intn(2) == 0 {
				return top2
			}
		}
	}

	return top1
}

// execute performs an RPC request with retry logic
func (mc *MultiClient) execute(ctx context.Context, op func(*Node) error) error {
	// Global rate limiting
	if err := mc.limiter.Wait(ctx); err != nil {
		return err
	}

	// Max attempts = number of nodes (capped at 3 to avoid long loops)
	attempts := len(mc.nodes)
	if attempts > 3 {
		attempts = 3
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		node := mc.pickBestNode()
		if node == nil {
			return ErrNoAvailableNodes
		}

		err := op(node)
		if err == nil {
			return nil
		}

		lastErr = err
		// If context is canceled, don't retry
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		// If failed, the node score will automatically decrease via RecordMetric
		// pickBestNode might select a different node in next attempt
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
