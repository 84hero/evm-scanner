package chain

import (
	"sync"
	"time"
)

// Preset defines the default behavior parameters for a chain
type Preset struct {
	ChainID   string
	BlockTime time.Duration // Average block time (affects polling interval)
	ReorgSafe uint64        // Recommended safety confirmations
	BatchSize uint64        // Recommended scan batch size
	Endpoint  string        // (Optional) Default public RPC
}

var (
	registry = make(map[string]Preset)
	mu       sync.RWMutex
)

// Register adds a new chain preset to the global registry.
func Register(name string, p Preset) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = p
}

// Get retrieves a preset configuration from the registry by its name.
func Get(name string) (Preset, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// Built-in presets
func init() {
	Register("eth-mainnet", Preset{
		ChainID:   "1",
		BlockTime: 12 * time.Second,
		ReorgSafe: 12,
		BatchSize: 100,
	})

	Register("bsc-mainnet", Preset{
		ChainID:   "56",
		BlockTime: 3 * time.Second,
		ReorgSafe: 15, // BSC reorgs are relatively frequent
		BatchSize: 200,
	})

	Register("polygon-mainnet", Preset{
		ChainID:   "137",
		BlockTime: 2 * time.Second,
		ReorgSafe: 32, // Polygon recommends deeper confirmations
		BatchSize: 200,
	})
}
