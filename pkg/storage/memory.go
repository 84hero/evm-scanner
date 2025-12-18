package storage

import (
	"sync"
)

// Persistence defines the interface for saving scanner progress
type Persistence interface {
	// LoadCursor reads the last scanned block height
	// key: task identifier (e.g., "erc20-scanner-v1")
	LoadCursor(key string) (uint64, error)

	// SaveCursor saves the current block height
	SaveCursor(key string, height uint64) error

	// Close releases resources
	Close() error
}

// MemoryStore is a simple in-memory implementation (Note: data lost on restart, for testing/temp tasks only)
type MemoryStore struct {
	data   map[string]uint64
	prefix string
	mu     sync.RWMutex
}

func NewMemoryStore(prefix string) *MemoryStore {
	return &MemoryStore{
		data:   make(map[string]uint64),
		prefix: prefix,
	}
}

func (m *MemoryStore) LoadCursor(key string) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[m.prefix+key], nil
}

func (m *MemoryStore) SaveCursor(key string, height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[m.prefix+key] = height
	return nil
}

func (m *MemoryStore) Close() error {
	return nil
}