package rpc

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EthClient abstracts the underlying ethclient.Client implementation for easier mocking/testing
type EthClient interface {
	ChainID(ctx context.Context) (*big.Int, error)
	BlockNumber(ctx context.Context) (uint64, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)
	Close()
}

// Client defines the minimal set of RPC methods required by the Scanner.
// This allows for mocking the client in tests or implementing multi-node load balancing.
type Client interface {
	// ChainID retrieves the chain ID
	ChainID(ctx context.Context) (*big.Int, error)
	
	// BlockNumber retrieves the latest block height
	BlockNumber(ctx context.Context) (uint64, error)
	
	// HeaderByNumber retrieves a block header (used for fast Bloom Filter checks)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	
	// BlockByNumber retrieves a full block (used for native transfer scanning)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	
	// FilterLogs retrieves logs (used for ERC20 scanning)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	
	// CodeAt checks contract code (used for safety validation)
	CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)
	
	// Close closes the connection
	Close()
}