package chain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRegistry(t *testing.T) {
	// 1. Test Built-in
	p, ok := Get("eth-mainnet")
	assert.True(t, ok)
	assert.Equal(t, "1", p.ChainID)

	// 2. Test Custom Register
	Register("my-test-chain", Preset{
		ChainID:   "123",
		BlockTime: 5 * time.Second,
	})

	p2, ok := Get("my-test-chain")
	assert.True(t, ok)
	assert.Equal(t, "123", p2.ChainID)
	assert.Equal(t, 5*time.Second, p2.BlockTime)

	// 3. Test Unknown
	_, ok = Get("unknown-chain")
	assert.False(t, ok)
}
