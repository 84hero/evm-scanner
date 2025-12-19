package storage

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements the Persistence interface using Redis as a backend.
type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewRedisStore initializes Redis storage
// addr: e.g., "localhost:6379"
// prefix: Key prefix (e.g., "evm_scanner:"). Final Key is prefix + task_key
func NewRedisStore(addr, password string, db int, prefix string) (*RedisStore, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	if prefix == "" {
		prefix = "scanner:"
	}

	return &RedisStore{
		client: rdb,
		prefix: prefix,
	}, nil
}

// LoadCursor retrieves the last scanned block height from Redis
func (r *RedisStore) LoadCursor(key string) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fullKey := r.prefix + key

	val, err := r.client.Get(ctx, fullKey).Uint64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return val, nil
}

// SaveCursor updates the last scanned block height in Redis
func (r *RedisStore) SaveCursor(key string, height uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fullKey := r.prefix + key

	// Set value with no expiration (0)
	return r.client.Set(ctx, fullKey, height, 0).Err()
}

// Close closes the Redis client connection
func (r *RedisStore) Close() error {
	return r.client.Close()
}
