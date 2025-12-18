package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/84hero/evm-scanner/pkg/chain"
	"github.com/84hero/evm-scanner/pkg/config"
	"github.com/84hero/evm-scanner/pkg/decoder"
	"github.com/84hero/evm-scanner/pkg/rpc"
	"github.com/84hero/evm-scanner/pkg/scanner"
	"github.com/84hero/evm-scanner/pkg/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// USDT ABI fragment (Transfer event only)
const usdtABI = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`

func main() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))

	// [Feature 1: Custom Chain Presets]
	// Suppose we are scanning a private chain "my-chain"
	chain.Register("my-chain", chain.Preset{
		ChainID:   "999",
		BlockTime: 1 * time.Second,
		ReorgSafe: 1,
		BatchSize: 500,
	})

	// 1. Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Crit("Failed to load config", "err", err)
	}

	// [Feature 1: Use presets to fill default values]
	// If BatchSize is not specified in config, use the chain's default
	if preset, ok := chain.Get(cfg.Scanner.ChainID); ok {
		if cfg.Scanner.BatchSize == 0 {
			cfg.Scanner.BatchSize = preset.BatchSize
		}
		if cfg.Scanner.Confirmations == 0 {
			cfg.Scanner.Confirmations = preset.ReorgSafe
		}
		log.Info("Loaded chain preset", "chain", cfg.Scanner.ChainID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize RPC
	client, err := rpc.NewClient(ctx, cfg.RPC, 10)
	if err != nil {
		log.Crit("Failed to init client", "err", err)
	}
	defer client.Close()

	// 3. Initialize Decoder
	usdtDecoder, err := decoder.NewFromJSON(usdtABI)
	if err != nil {
		log.Crit("Failed to init decoder", "err", err)
	}

	// 4. Build filter
	usdtAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	transferTopic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	
	filter := scanner.NewFilter().
		AddContract(usdtAddress).
		SetTopic(0, transferTopic)

	// 5. Configure Storage [Feature 2: Multiple Storage Engine Support]
	var store storage.Persistence
	
	// Storage Prefix (Namespace): Prioritize storage_prefix from config, otherwise use Project name
	storePrefix := cfg.Scanner.StoragePrefix
	if storePrefix == "" {
		storePrefix = cfg.Project + "_"
	}
	
	if dbURL := os.Getenv("PG_URL"); dbURL != "" {
		// PostgreSQL
		pgStore, err := storage.NewPostgresStore(dbURL, storePrefix)
		if err != nil {
			log.Crit("Failed to connect to Postgres", "err", err)
		}
		store = pgStore
		log.Info("Using PostgreSQL storage", "table_prefix", storePrefix)

	} else if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		// Redis
		// e.g., "localhost:6379"
		redisStore, err := storage.NewRedisStore(redisAddr, "", 0, storePrefix)
		if err != nil {
			log.Crit("Failed to connect to Redis", "err", err)
		}
		store = redisStore
		log.Info("Using Redis storage", "key_prefix", storePrefix)

	} else {
		// Memory (Default)
		store = storage.NewMemoryStore(storePrefix)
		log.Info("Using Memory storage (data lost on restart)", "internal_prefix", storePrefix)
	}

	// 6. Initialize Scanner
	scanCfg := scanner.Config{
		ChainID:       cfg.Scanner.ChainID,
		StartBlock:    cfg.Scanner.StartBlock,
		ForceStart:    cfg.Scanner.ForceStart,
		Rewind:        cfg.Scanner.Rewind,
		CursorRewind:  cfg.Scanner.CursorRewind,
		BatchSize:     cfg.Scanner.BatchSize,
		Interval:      cfg.Scanner.Interval,
		ReorgSafe:     cfg.Scanner.Confirmations, // Using merged preset values
		UseBloom:      cfg.Scanner.UseBloom,
	}

	s := scanner.New(client, store, scanCfg, filter)

	// 7. Set handle callback (Processor Layer)
	s.SetHandler(func(ctx context.Context, logs []types.Log) error {
		// [Logic: Local processing]
		for _, l := range logs {
			decoded, err := usdtDecoder.Decode(l)
			if err != nil {
				log.Error("Failed to decode log", "tx", l.TxHash.Hex(), "err", err)
				continue
			}

			// Print human-readable data
			fmt.Printf(" [Event] %s | Block: %d | From: %v | To: %v | Value: %v\n", 
				decoded.Name,
				l.BlockNumber,
				decoded.Inputs["from"],
				decoded.Inputs["to"],
				decoded.Inputs["value"],
			)
		}
		return nil
	})

	// 8. Start
	go func() {
		if err := s.Start(ctx); err != nil {
			log.Error("Scanner stopped", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}