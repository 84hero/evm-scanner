package scanner

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPersistence
type MockStore struct {
	mock.Mock
}

func (m *MockStore) LoadCursor(key string) (uint64, error) {
	args := m.Called(key)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockStore) SaveCursor(key string, height uint64) error {
	args := m.Called(key, height)
	return args.Error(0)
}

func (m *MockStore) Close() error {
	return m.Called().Error(0)
}

// MockRPC implements rpc.Client
type MockRPC struct {
	mock.Mock
}

func (m *MockRPC) ChainID(ctx context.Context) (*big.Int, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *MockRPC) BlockNumber(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockRPC) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	args := m.Called(ctx, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Header), args.Error(1)
}

func (m *MockRPC) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	args := m.Called(ctx, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Block), args.Error(1)
}

func (m *MockRPC) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Log), args.Error(1)
}

func (m *MockRPC) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	args := m.Called(ctx, account, blockNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockRPC) Close() {
	m.Called()
}

func TestDetermineStartBlock(t *testing.T) {
	store := new(MockStore)
	client := new(MockRPC)
	
	// Case 1: Force Start
	s := New(client, store, Config{ForceStart: true, StartBlock: 100}, nil)
	start, err := s.DetermineStartBlockForTest(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), start)

	// Case 2: Resume from Store (No Rewind)
	s = New(client, store, Config{ChainID: "eth"}, nil)
	store.On("LoadCursor", "eth").Return(uint64(500), nil).Once()
	start, _ = s.DetermineStartBlockForTest(context.Background())
	assert.Equal(t, uint64(500), start)

	// Case 3: Resume with Cursor Rewind
	s = New(client, store, Config{ChainID: "eth", CursorRewind: 10}, nil)
	store.On("LoadCursor", "eth").Return(uint64(500), nil).Once()
	start, _ = s.DetermineStartBlockForTest(context.Background())
	assert.Equal(t, uint64(490), start)
}

func TestDetermineStartBlock_Rewind(t *testing.T) {
	store := new(MockStore)
	client := new(MockRPC)

	// Case 4: No cursor, Rewind from Head
	// Config: Rewind = 100
	s := New(client, store, Config{ChainID: "eth", Rewind: 100}, nil)
	
	// Mock: LoadCursor -> 0 (Not found)
	store.On("LoadCursor", "eth").Return(uint64(0), nil).Once()
	// Mock: BlockNumber -> 1000
	client.On("BlockNumber", mock.Anything).Return(uint64(1000), nil).Once()

	start, err := s.DetermineStartBlockForTest(context.Background())
	assert.NoError(t, err)
	// Expected: 1000 - 100 = 900
	assert.Equal(t, uint64(900), start)
}

func TestScanner_Start_Errors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := new(MockStore)
	client := new(MockRPC)

	s := New(client, store, Config{
		ChainID: "eth", 
		Interval: 1 * time.Millisecond,
		ReorgSafe: 0,
		BatchSize: 1,
	}, NewFilter())

	// 1. StartBlock -> 100
	store.On("LoadCursor", "eth").Return(uint64(100), nil)

	// 2. Ticker 1: BlockNumber fails
	client.On("BlockNumber", mock.Anything).Return(uint64(0), assert.AnError).Once()

	// 3. Ticker 2: BlockNumber succeeds (102), but ScanRange fails
	client.On("BlockNumber", mock.Anything).Return(uint64(102), nil) // Called multiple times probably
	// FilterLogs fails
	client.On("FilterLogs", mock.Anything, mock.Anything).Return(nil, assert.AnError).Once()

	// 4. Cancel to stop
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := s.Start(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDetermineStartBlock_RewindBoundary(t *testing.T) {
	store := new(MockStore)
	client := new(MockRPC)
	// Head (50) < Rewind (100) -> Should return 0
	s := New(client, store, Config{ChainID: "eth", Rewind: 100}, nil)
	store.On("LoadCursor", "eth").Return(uint64(0), nil).Once()
	client.On("BlockNumber", mock.Anything).Return(uint64(50), nil).Once()

	start, err := s.DetermineStartBlockForTest(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), start)
}

func TestScanRange_BloomHit(t *testing.T) {
	store := new(MockStore)
	client := new(MockRPC)
	addr := common.HexToAddress("0x1234")
	filter := NewFilter().AddContract(addr)

	s := New(client, store, Config{UseBloom: true, BatchSize: 1}, filter)

	// 1. Mock Header with Bloom that MATCHES the address
	bloom := types.Bloom{}
	bloom.Add(addr.Bytes())
	header := &types.Header{Bloom: bloom}
	client.On("HeaderByNumber", mock.Anything, big.NewInt(100)).Return(header, nil)

	// 2. Expect FilterLogs to be called because Bloom HIT
	client.On("FilterLogs", mock.Anything, mock.Anything).Return([]types.Log{}, nil).Once()

	err := s.ScanRangeForTest(context.Background(), 100, 100)
	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestScanRange_BloomOptimization(t *testing.T) {
	store := new(MockStore)
	client := new(MockRPC)
	filter := NewFilter().AddContract(common.HexToAddress("0x1234"))

	s := New(client, store, Config{UseBloom: true, BatchSize: 1}, filter)

	// Mock Header with EMPTY bloom
	header := &types.Header{Bloom: types.Bloom{}}
	client.On("HeaderByNumber", mock.Anything, big.NewInt(100)).Return(header, nil)

	// Expect: scanRange should return nil WITHOUT calling FilterLogs (skipped)
	// If FilterLogs is called, the mock would panic because we didn't define expectation
	err := s.ScanRangeForTest(context.Background(), 100, 100)
	assert.NoError(t, err)
	client.AssertNotCalled(t, "FilterLogs")
}

func TestScanRange_Hit(t *testing.T) {
	store := new(MockStore)
	client := new(MockRPC)
	filter := NewFilter() // Empty filter matches everything

	s := New(client, store, Config{BatchSize: 10}, filter)
	
	// Mock FilterLogs return
	logs := []types.Log{{BlockNumber: 100}}
	client.On("FilterLogs", mock.Anything, mock.MatchedBy(func(q ethereum.FilterQuery) bool {
		return q.FromBlock.Int64() == 100 && q.ToBlock.Int64() == 105
	})).Return(logs, nil)

	// Handler
	handled := false
	s.SetHandler(func(ctx context.Context, l []types.Log) error {
		handled = true
		assert.Len(t, l, 1)
		return nil
	})

	err := s.ScanRangeForTest(context.Background(), 100, 105)
	assert.NoError(t, err)
	assert.True(t, handled)
}

// Expose private methods for testing
func (s *Scanner) DetermineStartBlockForTest(ctx context.Context) (uint64, error) {
	return s.determineStartBlock(ctx)
}

func (s *Scanner) ScanRangeForTest(ctx context.Context, from, to uint64) error {
	return s.scanRange(ctx, from, to)
}

func TestScanner_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := new(MockStore)
	client := new(MockRPC)
	
	// Mock process:
	// 1. determineStartBlock -> Start from 100
	store.On("LoadCursor", "eth").Return(uint64(100), nil)
	// 2. First ticker triggers:
	//    a. BlockNumber -> Chain height 105
	client.On("BlockNumber", mock.Anything).Return(uint64(105), nil)
	//    b. scanRange (100 to 102, assuming safe confirmation 3)
	//       105 - 3 = 102. So scan 100 to 102.
	client.On("FilterLogs", mock.Anything, mock.Anything).Return([]types.Log{}, nil)
	//    c. SaveCursor fails (should log and continue)
	store.On("SaveCursor", "eth", mock.Anything).Return(assert.AnError).Once()
	// Next iteration
	store.On("SaveCursor", "eth", mock.Anything).Return(nil)

	s := New(client, store, Config{
		ChainID: "eth", 
		Interval: 10 * time.Millisecond, 
		ReorgSafe: 3,
		BatchSize: 10,
	}, NewFilter())

	// Cancel after running for a while
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := s.Start(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}
