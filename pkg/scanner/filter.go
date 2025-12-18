package scanner

import (
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Filter defines the scanning rules for the scanner.
// It is used both for generating RPC request parameters and for local Bloom Filter checks.
type Filter struct {
	// Contracts is the list of contract addresses to listen to (Log.Address).
	// If empty, listens to all contracts.
	Contracts []common.Address

	// Topics is the list of event topics to listen to.
	// Maps to eth_getLogs topics parameter: [[A, B], [C], null, [D]]
	// Logical relation: (Topic0 in [A, B]) AND (Topic1 in [C])
	Topics [][]common.Hash
}

// NewFilter creates a new filter
func NewFilter() *Filter {
	return &Filter{
		Contracts: make([]common.Address, 0),
		Topics:    make([][]common.Hash, 0),
	}
}

// AddContract adds contract addresses to listen to
func (f *Filter) AddContract(addrs ...common.Address) *Filter {
	f.Contracts = append(f.Contracts, addrs...)
	return f
}

// SetTopic sets the topics at a specific position
// pos: 0-3 (0 is typically the event signature hash)
func (f *Filter) SetTopic(pos int, hashes ...common.Hash) *Filter {
	// Ensure storage space is sufficient
	if len(f.Topics) <= pos {
		// Expand
		newTopics := make([][]common.Hash, pos+1)
		copy(newTopics, f.Topics)
		f.Topics = newTopics
	}
	f.Topics[pos] = append(f.Topics[pos], hashes...)
	return f
}

// ToQuery converts the filter to go-ethereum standard query parameters
func (f *Filter) ToQuery(fromBlock, toBlock uint64) ethereum.FilterQuery {
	// Build query
	q := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: f.Contracts,
		Topics:    f.Topics,
	}
	return q
}

// IsHeavy determines if the filter is too complex for efficient local Bloom Filter checks.
// Rule of thumb: Bloom filter tends to saturate if topics at any position > 20 or contracts > 20.
func (f *Filter) IsHeavy() bool {
	if len(f.Contracts) > 20 {
		return true
	}
	for _, subTopics := range f.Topics {
		if len(subTopics) > 20 {
			return true
		}
	}
	return false
}

// MatchesBloom uses the local Bloom Filter to quickly check if a block might contain matching logs.
// Returns false if it definitely doesn't contain matching logs (Safe to Skip).
// Returns true if it might contain matching logs (Need to Fetch).
func (f *Filter) MatchesBloom(bloom types.Bloom) bool {
	// 1. Check contract addresses
	// Logic: If Contracts is set, Block Bloom must contain at least one of them.
	if len(f.Contracts) > 0 {
		found := false
		for _, addr := range f.Contracts {
			if bloom.Test(addr.Bytes()) {
				found = true
				break
			}
		}
		if !found {
			return false // No contracts in Bloom, skip
		}
	}

	// 2. Check Topics
	// Logic: For each set Topic position, Block Bloom must contain at least one hash from that position.
	for _, subTopics := range f.Topics {
		if len(subTopics) == 0 {
			continue // No filter for this position (wildcard), skip check
		}

		found := false
		for _, hash := range subTopics {
			if bloom.Test(hash.Bytes()) {
				found = true
				break
			}
		}
		if !found {
			return false // None of the candidate topics in Bloom, AND condition not met, skip
		}
	}

	return true
}