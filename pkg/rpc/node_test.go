package rpc

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestNewNode(t *testing.T) {
	ctx := context.Background()
	// Fails to dial invalid URL
	_, err := NewNode(ctx, NodeConfig{URL: "invalid", Priority: 10})
	assert.Error(t, err)
}

func TestNode_ProxyMethods(t *testing.T) {
	ctx := context.Background()
	mockEth := new(MockEthClient)
	node := NewNodeWithClient(NodeConfig{URL: "test", Priority: 10}, mockEth)

	// 1. BlockNumber
	mockEth.On("BlockNumber", ctx).Return(uint64(100), nil).Once()
	h, err := node.BlockNumber(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), h)

	// 2. ChainID
	mockEth.On("ChainID", ctx).Return(big.NewInt(1), nil).Once()
	id, err := node.ChainID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), id.Int64())

	// 3. HeaderByNumber
	mockEth.On("HeaderByNumber", ctx, big.NewInt(100)).Return(&types.Header{}, nil).Once()
	_, err = node.HeaderByNumber(ctx, big.NewInt(100))
	assert.NoError(t, err)

	// 4. BlockByNumber
	mockEth.On("BlockByNumber", ctx, big.NewInt(100)).Return(&types.Block{}, nil).Once()
	_, err = node.BlockByNumber(ctx, big.NewInt(100))
	assert.NoError(t, err)

	// 5. FilterLogs
	mockEth.On("FilterLogs", ctx, ethereum.FilterQuery{}).Return([]types.Log{}, nil).Once()
	_, err = node.FilterLogs(ctx, ethereum.FilterQuery{})
	assert.NoError(t, err)

	// 6. CodeAt
	addr := common.HexToAddress("0x1")
	mockEth.On("CodeAt", ctx, addr, big.NewInt(100)).Return([]byte{0x1}, nil).Once()
	_, err = node.CodeAt(ctx, addr, big.NewInt(100))
	assert.NoError(t, err)

	// 7. Close
	mockEth.On("Close").Once()
	node.Close()
}
