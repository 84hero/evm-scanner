package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/84hero/evm-scanner/pkg/chain"
	"github.com/84hero/evm-scanner/pkg/config"
	"github.com/84hero/evm-scanner/pkg/decoder"
	"github.com/84hero/evm-scanner/pkg/rpc"
	"github.com/84hero/evm-scanner/pkg/scanner"
	"github.com/84hero/evm-scanner/pkg/sink"
	"github.com/84hero/evm-scanner/pkg/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/spf13/viper"
)

// --- Configuration Structs ---

type AppConfig struct {
	Filters []CLIFilterConfig `mapstructure:"filters"`
	Outputs OutputsConfig     `mapstructure:"outputs"`
	Webhook WebhookConfig     `mapstructure:"webhook"`
}

type OutputsConfig struct {
	Webhook  WebhookOutputConfig  `mapstructure:"webhook"`
	File     FileOutputConfig     `mapstructure:"file"`
	Console  ConsoleOutputConfig  `mapstructure:"console"`
	Postgres PostgresOutputConfig `mapstructure:"postgres"`
	Redis    RedisOutputConfig    `mapstructure:"redis"`
	Kafka    KafkaOutputConfig    `mapstructure:"kafka"`
	RabbitMQ RabbitMQOutputConfig `mapstructure:"rabbitmq"`
}

type WebhookOutputConfig struct {
	Enabled    bool        `mapstructure:"enabled"`
	URL        string      `mapstructure:"url"`
	Secret     string      `mapstructure:"secret"`
	Retry      RetryConfig `mapstructure:"retry"`
	Async      bool        `mapstructure:"async"`
	BufferSize int         `mapstructure:"buffer_size"`
	Workers    int         `mapstructure:"workers"`
}

type WebhookConfig = WebhookOutputConfig

type FileOutputConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type ConsoleOutputConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type PostgresOutputConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
	Table   string `mapstructure:"table"`
}

type RedisOutputConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Key      string `mapstructure:"key"`
	Mode     string `mapstructure:"mode"`
}

type KafkaOutputConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	Brokers  []string `mapstructure:"brokers"`
	Topic    string   `mapstructure:"topic"`
	User     string   `mapstructure:"user"`
	Password string   `mapstructure:"password"`
}

type RabbitMQOutputConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	URL        string `mapstructure:"url"`
	Exchange   string `mapstructure:"exchange"`
	RoutingKey string `mapstructure:"routing_key"`
	QueueName  string `mapstructure:"queue_name"`
	Durable    bool   `mapstructure:"durable"`
}

type RetryConfig struct {
	MaxAttempts    int           `mapstructure:"max_attempts"`
	InitialBackoff time.Duration `mapstructure:"initial_backoff"`
	MaxBackoff     time.Duration `mapstructure:"max_backoff"`
}

type CLIFilterConfig struct {
	Description string     `mapstructure:"description"`
	Contracts   []string   `mapstructure:"contracts"`
	Topics      [][]string `mapstructure:"topics"`
	ABI         string     `mapstructure:"abi"`
}

// --- Helper Functions ---

func loadAppConfig(path string) (*AppConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func initFilters(configs []CLIFilterConfig) (*scanner.Filter, map[common.Hash]*decoder.ABIWrapper) {
	filter := scanner.NewFilter()
	decoders := make(map[common.Hash]*decoder.ABIWrapper)
	for _, f := range configs {
		for _, c := range f.Contracts {
			if common.IsHexAddress(c) {
				filter.AddContract(common.HexToAddress(c))
			}
		}
		for i, topicGroup := range f.Topics {
			var hashes []common.Hash
			for _, t := range topicGroup {
				hashes = append(hashes, common.HexToHash(t))
			}
			filter.SetTopic(i, hashes...)
		}
		if f.ABI != "" {
			dec, _ := decoder.NewFromJSON(f.ABI)
			if dec != nil && len(f.Topics) > 0 && len(f.Topics[0]) > 0 {
				for _, sig := range f.Topics[0] {
					decoders[common.HexToHash(sig)] = dec
				}
			}
		}
	}
	return filter, decoders
}

func initOutputs(appCfg *AppConfig) []sink.Output {
	var outputs []sink.Output

	// Webhook
	wh := appCfg.Outputs.Webhook
	if !wh.Enabled && appCfg.Webhook.URL != "" {
		wh = appCfg.Webhook
		wh.Enabled = true
	}
	if wh.Enabled {
		outputs = append(outputs, sink.NewWebhookOutput(wh.URL, wh.Secret, wh.Retry.MaxAttempts, wh.Retry.InitialBackoff.String(), wh.Retry.MaxBackoff.String(), wh.Async, wh.BufferSize, wh.Workers))
	}

	// File
	if appCfg.Outputs.File.Enabled {
		if fo, err := sink.NewFileOutput(appCfg.Outputs.File.Path); err == nil {
			outputs = append(outputs, fo)
		}
	}

	// Console
	if appCfg.Outputs.Console.Enabled {
		outputs = append(outputs, sink.NewConsoleOutput())
	}

	// Postgres
	if appCfg.Outputs.Postgres.Enabled {
		if po, err := sink.NewPostgresOutput(appCfg.Outputs.Postgres.URL, appCfg.Outputs.Postgres.Table); err == nil {
			outputs = append(outputs, po)
		}
	}

	// Redis
	if appCfg.Outputs.Redis.Enabled {
		if ro, err := sink.NewRedisOutput(appCfg.Outputs.Redis.Addr, appCfg.Outputs.Redis.Password, appCfg.Outputs.Redis.DB, appCfg.Outputs.Redis.Key, appCfg.Outputs.Redis.Mode); err == nil {
			outputs = append(outputs, ro)
		}
	}

	// Kafka
	if appCfg.Outputs.Kafka.Enabled {
		if ko, err := sink.NewKafkaOutput(appCfg.Outputs.Kafka.Brokers, appCfg.Outputs.Kafka.Topic, appCfg.Outputs.Kafka.User, appCfg.Outputs.Kafka.Password); err == nil {
			outputs = append(outputs, ko)
		}
	}

	// RabbitMQ
	if appCfg.Outputs.RabbitMQ.Enabled {
		if ro, err := sink.NewRabbitMQOutput(appCfg.Outputs.RabbitMQ.URL, appCfg.Outputs.RabbitMQ.Exchange, appCfg.Outputs.RabbitMQ.RoutingKey, appCfg.Outputs.RabbitMQ.QueueName, appCfg.Outputs.RabbitMQ.Durable); err == nil {
			outputs = append(outputs, ro)
		}
	}

	return outputs
}

func main() {
	if err := Run(context.Background()); err != nil && err != context.Canceled {
		log.Crit("Application failed", "err", err)
		os.Exit(1)
	}
}

// Run is the testable entry point of the CLI application
func Run(ctx context.Context) error {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))

	coreConfigFile := os.Getenv("CONFIG_FILE")
	if coreConfigFile == "" {
		coreConfigFile = "config.yaml"
	}
	coreCfg, err := config.Load(coreConfigFile)
	if err != nil {
		return err
	}

	// Setup Logger
	logLevel := log.LevelInfo
	if coreCfg.Log.Level == "debug" {
		logLevel = log.LevelDebug
	} else if coreCfg.Log.Level == "warn" {
		logLevel = log.LevelWarn
	} else if coreCfg.Log.Level == "error" {
		logLevel = log.LevelError
	}

	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, logLevel, true)))

	appConfigFile := os.Getenv("APP_CONFIG_FILE")
	if appConfigFile == "" {
		appConfigFile = "app.yaml"
	}
	appCfg, err := loadAppConfig(appConfigFile)
	if err != nil {
		log.Warn("Failed to load app config", "err", err)
		appCfg = &AppConfig{}
	}

	// Chain Presets
	if preset, ok := chain.Get(coreCfg.Scanner.ChainID); ok {
		if coreCfg.Scanner.BatchSize == 0 {
			coreCfg.Scanner.BatchSize = preset.BatchSize
		}
		if coreCfg.Scanner.Confirmations == 0 {
			coreCfg.Scanner.Confirmations = preset.ReorgSafe
		}
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Components
	client, err := rpc.NewClient(runCtx, coreCfg.RPC, 20)
	if err != nil {
		return err
	}
	defer client.Close()

	filter, decoders := initFilters(appCfg.Filters)
	outputs := initOutputs(appCfg)
	defer func() {
		for _, o := range outputs {
			o.Close()
		}
	}()

	// Storage
	var store storage.Persistence
	storePrefix := coreCfg.Scanner.StoragePrefix
	if storePrefix == "" {
		storePrefix = coreCfg.Project + "_"
	}
	if dbURL := os.Getenv("PG_URL"); dbURL != "" {
		store, _ = storage.NewPostgresStore(dbURL, storePrefix)
	} else if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		store, _ = storage.NewRedisStore(redisAddr, "", 0, storePrefix)
	} else {
		store = storage.NewMemoryStore(storePrefix)
	}

	// Scanner
	scanCfg := scanner.Config{
		ChainID:      coreCfg.Scanner.ChainID,
		StartBlock:   coreCfg.Scanner.StartBlock,
		ForceStart:   coreCfg.Scanner.ForceStart,
		Rewind:       coreCfg.Scanner.Rewind,
		CursorRewind: coreCfg.Scanner.CursorRewind,
		BatchSize:    coreCfg.Scanner.BatchSize,
		Interval:     coreCfg.Scanner.Interval,
		ReorgSafe:    coreCfg.Scanner.Confirmations,
		UseBloom:     coreCfg.Scanner.UseBloom,
	}

	s := scanner.New(client, store, scanCfg, filter)
	s.SetHandler(func(ctx context.Context, logs []types.Log) error {
		var decodedLogs []sink.DecodedLog
		for _, l := range logs {
			dl := sink.DecodedLog{Log: l}
			if len(l.Topics) > 0 {
				if dec, ok := decoders[l.Topics[0]]; ok {
					if res, err := dec.Decode(l); err == nil {
						dl.DecodedData = res
						dl.EventName = res.Name
					}
				}
			}
			decodedLogs = append(decodedLogs, dl)
		}
		var wg sync.WaitGroup
		for _, out := range outputs {
			wg.Add(1)
			go func(o sink.Output) {
				defer wg.Done()
				_ = o.Send(ctx, decodedLogs)
			}(out)
		}
		wg.Wait()
		return nil
	})

	go func() {
		if err := s.Start(runCtx); err != nil {
			log.Error("Scanner failed", "err", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Info("Shutting down...")
	case <-ctx.Done():
	}

	cancel()
	time.Sleep(500 * time.Millisecond)
	return nil
}
