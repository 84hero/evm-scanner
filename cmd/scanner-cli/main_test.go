package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestCLI_LoadAppConfig(t *testing.T) {
	content := `
filters:
  - description: "test filter"
    contracts: ["0xdAC17F958D2ee523a2206206994597C13D831ec7"]
outputs:
  console:
    enabled: true
  webhook:
    enabled: true
    url: "http://localhost"
    async: true
`
	tmpFile, _ := os.CreateTemp("", "app_*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(content)
	tmpFile.Close()

	cfg, err := loadAppConfig(tmpFile.Name())
	assert.NoError(t, err)
	assert.Len(t, cfg.Filters, 1)
	assert.True(t, cfg.Outputs.Console.Enabled)
	assert.True(t, cfg.Outputs.Webhook.Async)
}

func TestCLI_LoadAppConfig_Fail(t *testing.T) {
	_, err := loadAppConfig("non_existent.yaml")
	assert.Error(t, err)
}

func TestCLI_InitFilters(t *testing.T) {
	configs := []CLIFilterConfig{
		{
			Description: "Test",
			Contracts:   []string{"0xdAC17F958D2ee523a2206206994597C13D831ec7"},
			Topics:      [][]string{{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"}},
		},
	}

	filter, _ := initFilters(configs)
	assert.NotNil(t, filter)
	assert.True(t, common.IsHexAddress(configs[0].Contracts[0]))
}

func TestCLI_InitFilters_Empty(t *testing.T) {
	filter, decoders := initFilters([]CLIFilterConfig{})
	assert.NotNil(t, filter)
	assert.Empty(t, decoders)
}

func TestCLI_InitFilters_WithABI(t *testing.T) {
	configs := []CLIFilterConfig{
		{
			Description: "USDT",
			Contracts:   []string{"0xdAC17F958D2ee523a2206206994597C13D831ec7"},
			Topics:      [][]string{{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"}},
			ABI:         `[{"anonymous":false,"inputs":[],"name":"Transfer","type":"event"}]`,
		},
	}

	filter, decoders := initFilters(configs)
	assert.NotNil(t, filter)
	assert.Len(t, decoders, 1)
}

func TestCLI_InitOutputs_Empty(t *testing.T) {
	outputs := initOutputs(&AppConfig{})
	assert.Empty(t, outputs)
}

func TestCLI_InitOutputs_ConsoleFile(t *testing.T) {
	appCfg := &AppConfig{
		Outputs: OutputsConfig{
			Console: ConsoleOutputConfig{Enabled: true},
			File:    FileOutputConfig{Enabled: true, Path: "/tmp/test.log"},
		},
	}
	defer os.Remove("/tmp/test.log")

	outputs := initOutputs(appCfg)
	assert.GreaterOrEqual(t, len(outputs), 1)

	foundConsole := false
	for _, o := range outputs {
		if o.Name() == "console" {
			foundConsole = true
		}
	}
	assert.True(t, foundConsole)
}

func TestCLI_Run(t *testing.T) {
	coreCfg := `
project: "test"
rpc_nodes: [{url: "invalid-scheme://", priority: 1}]
`
	appCfg := `
filters: []
outputs: {console: {enabled: true}}
`
	coreFile, _ := os.CreateTemp("", "core_*.yaml")
	appFile, _ := os.CreateTemp("", "app_*.yaml")
	defer os.Remove(coreFile.Name())
	defer os.Remove(appFile.Name())
	
	coreFile.WriteString(coreCfg)
	appFile.WriteString(appCfg)
	coreFile.Close()
	appFile.Close()

	os.Setenv("CONFIG_FILE", coreFile.Name())
	os.Setenv("APP_CONFIG_FILE", appFile.Name())
	defer os.Unsetenv("CONFIG_FILE")
	defer os.Unsetenv("APP_CONFIG_FILE")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := Run(ctx)
	assert.Error(t, err)
}