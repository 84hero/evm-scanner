package scanner

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestFilter_Builder(t *testing.T) {
	f := NewFilter()

	addr1 := common.HexToAddress("0x1111")
	addr2 := common.HexToAddress("0x2222")

	f.AddContract(addr1)
	f.AddContract(addr2)
	assert.Len(t, f.Contracts, 2)

	topic := common.HexToHash("0xaaaa")
	f.SetTopic(0, topic)
	assert.Len(t, f.Topics, 1)
	assert.Len(t, f.Topics[0], 1)

	// Test Expansion (SetTopic at index 2, should auto-fill index 1)
	f.SetTopic(2, topic)
	assert.Len(t, f.Topics, 3)
	assert.Len(t, f.Topics[1], 0) // Empty
	assert.Len(t, f.Topics[2], 1)

	// ToQuery
	q := f.ToQuery(100, 200)
	assert.Equal(t, int64(100), q.FromBlock.Int64())
	assert.Equal(t, int64(200), q.ToBlock.Int64())
	assert.Len(t, q.Addresses, 2)
}

func TestFilter_IsHeavy(t *testing.T) {
	f := NewFilter()
	assert.False(t, f.IsHeavy())

	// Add many contracts
	for i := 0; i < 21; i++ {
		f.AddContract(common.HexToAddress("0x1"))
	}
	assert.True(t, f.IsHeavy())

	f = NewFilter()
	// Add many topics
	hashes := make([]common.Hash, 21)
	f.SetTopic(0, hashes...)
	assert.True(t, f.IsHeavy())
}

func TestFilter_MatchesBloom(t *testing.T) {
	// 1. Setup Bloom that contains addr1 and topic1
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	topic1 := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	bloom := types.Bloom{}
	bloom.Add(addr1.Bytes())
	bloom.Add(topic1.Bytes())

	// 2. Test Contract Match
	f1 := NewFilter().AddContract(addr1)
	assert.True(t, f1.MatchesBloom(bloom))

	f2 := NewFilter().AddContract(common.HexToAddress("0x2222222222222222222222222222222222222222"))
	assert.False(t, f2.MatchesBloom(bloom))

	// 3. Test Topic Match
	f3 := NewFilter().SetTopic(0, topic1)
	assert.True(t, f3.MatchesBloom(bloom))

	f4 := NewFilter().SetTopic(0, common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))
	assert.False(t, f4.MatchesBloom(bloom))

	// 4. Test Combined (AND logic)
	f5 := NewFilter().AddContract(addr1).SetTopic(0, topic1)
	assert.True(t, f5.MatchesBloom(bloom))

	// Contract matches, but Topic doesn't -> False
	f6 := NewFilter().AddContract(addr1).SetTopic(0, common.HexToHash("0xbb"))
	assert.False(t, f6.MatchesBloom(bloom))
}
