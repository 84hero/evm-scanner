package scanner

import (
	"context"
	"math/big"
	"time"

	"github.com/84hero/evm-scanner/pkg/rpc"
	"github.com/84hero/evm-scanner/pkg/storage"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type Config struct {
	ChainID string
	// Startup strategy
	StartBlock   uint64
	ForceStart   bool
	Rewind       uint64
	CursorRewind uint64 // Safety rewind from saved cursor

	BatchSize uint64
	Interval  time.Duration
	ReorgSafe uint64
	UseBloom  bool
}

type Handler func(ctx context.Context, logs []types.Log) error

type Scanner struct {
	client  rpc.Client
	store   storage.Persistence
	config  Config
	filter  *Filter
	handler Handler
}

func New(client rpc.Client, store storage.Persistence, cfg Config, filter *Filter) *Scanner {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.Interval == 0 {
		cfg.Interval = 3 * time.Second
	}
	return &Scanner{
		client: client,
		store:  store,
		config: cfg,
		filter: filter,
	}
}

// SetHandler sets the callback function to be called when logs are received
func (s *Scanner) SetHandler(h Handler) {
	s.handler = h
}

// Start begins the scanning loop (blocks until context is cancelled)
func (s *Scanner) Start(ctx context.Context) error {
	// 1. Determine starting block height
	// Note: determineStartBlock might call RPC to get latest block (if using Rewind logic)
	currentBlock, err := s.determineStartBlock(ctx)
	if err != nil {
		return err
	}
	log.Info("Scanner started", "start_block", currentBlock, "chain_id", s.config.ChainID)

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// 2. Get latest block number from chain
			head, err := s.client.BlockNumber(ctx)
			if err != nil {
				log.Error("Failed to get block number", "err", err)
				continue
			}

			// Calculate safe height (latest height - confirmations)
			safeHead := head - s.config.ReorgSafe
			if safeHead < currentBlock {
				// No new blocks yet
				continue
			}

			// 3. Catch up loop
			for currentBlock <= safeHead {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				// Calculate end block for current batch
				endBlock := currentBlock + s.config.BatchSize - 1
				if endBlock > safeHead {
					endBlock = safeHead
				}

				// 4. Perform scanning
				err := s.scanRange(ctx, currentBlock, endBlock)
				if err != nil {
					log.Error("Scan range failed", "from", currentBlock, "to", endBlock, "err", err)
					// Wait a bit before retrying, but respect context
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(1 * time.Second):
					}
					break // Break inner loop, wait for next ticker
				}

				// 5. Update progress
				// Next start from endBlock + 1
				nextStart := endBlock + 1
				if err := s.store.SaveCursor(s.config.ChainID, nextStart); err != nil {
					log.Error("Failed to save cursor", "err", err)
				}

				currentBlock = nextStart
			}
		}
	}
}

func (s *Scanner) determineStartBlock(ctx context.Context) (uint64, error) {
	// Strategy 1: Force Start (highest priority)
	if s.config.ForceStart && s.config.StartBlock > 0 {
		log.Info("Start strategy: Force Start", "block", s.config.StartBlock)
		return s.config.StartBlock, nil
	}

	// Strategy 2: Resume from persistence
	saved, err := s.store.LoadCursor(s.config.ChainID)
	if err != nil {
		return 0, err
	}
	if saved > 0 {
		start := saved
		if s.config.CursorRewind > 0 {
			if start > s.config.CursorRewind {
				start = start - s.config.CursorRewind
			} else {
				start = 0
			}
			log.Info("Start strategy: Resume from persistence with safety rewind", "saved", saved, "rewind", s.config.CursorRewind, "start", start)
		} else {
			log.Info("Start strategy: Resume from persistence", "block", saved)
		}
		return start, nil
	}

	// Strategy 3: Config StartBlock (not forced, used as default)
	if s.config.StartBlock > 0 {
		log.Info("Start strategy: Config StartBlock", "block", s.config.StartBlock)
		return s.config.StartBlock, nil
	}

	// Strategy 4: Dynamic Rewind
	// If no saved cursor and no StartBlock, start from N blocks before Head
	head, err := s.client.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}

	start := uint64(0)
	if head > s.config.Rewind {
		start = head - s.config.Rewind
	}
	log.Info("Start strategy: Rewind from Head", "head", head, "rewind", s.config.Rewind, "start", start)

	return start, nil
}

func (s *Scanner) scanRange(ctx context.Context, from, to uint64) error {
	// Strategy: Check if Bloom optimization should be used
	// If:
	// 1. Bloom optimization enabled
	// 2. Filter is not "heavy"
	// 3. Scan range is small (Bloom is most effective for single or few blocks)
	// For simplicity, we only use Bloom when BatchSize=1 or scanning single block
	// eth_getLogs is usually fast enough anyway.

	shouldCheckBloom := s.config.UseBloom && !s.filter.IsHeavy() && (to == from)

	if shouldCheckBloom {
		header, err := s.client.HeaderByNumber(ctx, big.NewInt(int64(from)))
		if err != nil {
			return err
		}
		// Local Bloom check
		if !s.filter.MatchesBloom(header.Bloom) {
			// Bloom says definitely not here, skip
			return nil
		}
		// Bloom says possibly here, continue to eth_getLogs
	}

	// Build eth_getLogs request
	query := s.filter.ToQuery(from, to)

	// Set range
	query.FromBlock = big.NewInt(int64(from))
	query.ToBlock = big.NewInt(int64(to))

	logs, err := s.client.FilterLogs(ctx, query)
	if err != nil {
		return err
	}

	if len(logs) > 0 && s.handler != nil {
		if err := s.handler(ctx, logs); err != nil {
			return err
		}
	}

	return nil
}
