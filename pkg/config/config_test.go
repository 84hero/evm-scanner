package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// 1. Normal load test
	content := `
project: "test-proj"
scanner:
  chain_id: "1"
  batch_size: 50
  interval: "1s"
rpc_nodes:
  - url: "http://localhost:8545"
    priority: 1
`
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "test-proj", cfg.Project)
	assert.Equal(t, uint64(50), cfg.Scanner.BatchSize)
	assert.Equal(t, 1*time.Second, cfg.Scanner.Interval)

	// 2. File not found test
	_, err = Load("non_existent_file.yaml")
	assert.Error(t, err)

	// 3. Invalid format test
	tmpFile2, _ := os.CreateTemp("", "invalid_*.yaml")
	_, err = tmpFile2.WriteString("invalid_yaml: [ unclosed bracket")
	assert.NoError(t, err)
	tmpFile2.Close()
	defer os.Remove(tmpFile2.Name())
	
	_, err = Load(tmpFile2.Name())
	assert.Error(t, err)
}

func TestLoad_Defaults(t *testing.T) {
	// Test default values: when batch_size and interval are not specified
	content := `
project: "defaults"
scanner:
  chain_id: "1"
`
	tmpFile, err := os.CreateTemp("", "config_defaults_*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	assert.NoError(t, err)
	
	// Verify default values (BatchSize=100, Interval=3s)
	assert.Equal(t, uint64(100), cfg.Scanner.BatchSize)
	assert.Equal(t, 3*time.Second, cfg.Scanner.Interval)
}

func TestLoad_EnvVars(t *testing.T) {
	// Create a config containing target keys (values can be empty or default for Viper to override)
	content := `
project: "default"
scanner:
  chain_id: "1"
  batch_size: 10
`
	tmpFile, err := os.CreateTemp("", "config_env_*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// Set environment variables
	os.Setenv("SCANNER_PROJECT", "env-project")
	os.Setenv("SCANNER_SCANNER_BATCH_SIZE", "999")
	defer func() {
		os.Unsetenv("SCANNER_PROJECT")
		os.Unsetenv("SCANNER_SCANNER_BATCH_SIZE")
	}()

	cfg, err := Load(tmpFile.Name())
	assert.NoError(t, err)

	// Verify environment variable overrides
	assert.Equal(t, "env-project", cfg.Project)
	assert.Equal(t, uint64(999), cfg.Scanner.BatchSize)
}