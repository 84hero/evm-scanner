package config

import (
	"strings"
	"time"

	"github.com/84hero/evm-scanner/pkg/rpc"
	"github.com/spf13/viper"
)

type Config struct {
	Project string        `mapstructure:"project"`
	Log     LogConfig     `mapstructure:"log"`
	Scanner ScannerConfig `mapstructure:"scanner"`
	RPC     []rpc.NodeConfig `mapstructure:"rpc_nodes"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // text, json
}

type ScannerConfig struct {
	ChainID   string        `mapstructure:"chain_id"`
	BatchSize uint64        `mapstructure:"batch_size"`
	Interval  time.Duration `mapstructure:"interval"`
	
	// Confirmations (ReorgSafeDepth): Protection at the scanning endpoint
	Confirmations uint64 `mapstructure:"confirmations"`

	// Startup strategy
	StartBlock uint64 `mapstructure:"start_block"`  // If > 0 and ForceStart=true, forces start from here
	ForceStart bool   `mapstructure:"force_start"`  // Whether to force override persistence records
	Rewind     uint64 `mapstructure:"start_rewind"` // If no saved cursor, start from Latest - Rewind
	CursorRewind uint64 `mapstructure:"cursor_rewind"` // If saved cursor exists, start from Cursor - CursorRewind (safety buffer)

	UseBloom bool `mapstructure:"use_bloom"`
	
	// StoragePrefix: Prefix for storage layer (e.g., PG table prefix or Redis Key prefix)
	StoragePrefix string `mapstructure:"storage_prefix"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("SCANNER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Set default values
	if cfg.Scanner.BatchSize == 0 {
		cfg.Scanner.BatchSize = 100
	}
	if cfg.Scanner.Interval == 0 {
		cfg.Scanner.Interval = 3 * time.Second
	}

	return &cfg, nil
}