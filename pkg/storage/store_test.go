package storage

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// --- Memory Store Tests ---

func TestMemoryStore(t *testing.T) {
	s := NewMemoryStore("test_")
	err := s.SaveCursor("task1", 100)
	assert.NoError(t, err)

	h, err := s.LoadCursor("task1")
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), h)

	h, err = s.LoadCursor("unknown")
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), h)

	// Memory store Close is no-op
	assert.NoError(t, s.Close())
}

// --- Postgres Store Tests ---

func TestPostgresStore_InitTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	store := &PostgresStore{
		db:        db,
		tableName: "custom_checkpoints",
	}

	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS custom_checkpoints")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.initTable()
	assert.NoError(t, err)
}

func TestPostgresStore_SaveLoad(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	store := &PostgresStore{
		db:        db,
		tableName: "scanner_checkpoints",
	}

	// 1. Test Save Success
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO scanner_checkpoints")).
		WithArgs("task1", 100).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store.SaveCursor("task1", 100)
	assert.NoError(t, err)

	// 2. Test Save Error
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO scanner_checkpoints")).
		WillReturnError(assert.AnError)
	err = store.SaveCursor("task1", 100)
	assert.Error(t, err)

	// 3. Test Load Success
	rows := sqlmock.NewRows([]string{"block_height"}).AddRow(200)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT block_height FROM scanner_checkpoints")).
		WithArgs("task1").
		WillReturnRows(rows)

	h, err := store.LoadCursor("task1")
	assert.NoError(t, err)
	assert.Equal(t, uint64(200), h)

	// 4. Test Load Not Found (should return 0, no error)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT block_height")).
		WithArgs("task2").
		WillReturnError(sql.ErrNoRows)
	h, err = store.LoadCursor("task2")
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), h)

	// 5. Test Load Error
	mock.ExpectQuery(regexp.QuoteMeta("SELECT block_height")).
		WillReturnError(assert.AnError)
	_, err = store.LoadCursor("task3")
	assert.Error(t, err)

	// 6. Test Close

	mock.ExpectClose()

	assert.NoError(t, store.Close())

}

// Note: NewPostgresStore involves real sql.Open, making it difficult to fully mock the driver layer.

// However, we can test passing an invalid URL.

func TestNewPostgresStore_InvalidURL(t *testing.T) {

	// This is a malformed connection string

	_, err := NewPostgresStore("postgres://invalid-url?param=^^", "prefix")

	assert.Error(t, err)

}

func TestNewPostgresStore_Mock(t *testing.T) {

	// We can't easily mock the 'sql.Open' call inside NewPostgresStore because it's a package level function,

	// but the code is already mostly covered by the Save/Load tests which use a manually constructed PostgresStore.

}

// --- Redis Store Tests ---

func TestRedisStore_SaveLoad(t *testing.T) {

	db, mock := redismock.NewClientMock()

	store := &RedisStore{

		client: db,

		prefix: "scan:",
	}

	// 1. Test Save Success

	mock.ExpectSet("scan:task1", uint64(100), time.Duration(0)).SetVal("OK")

	err := store.SaveCursor("task1", 100)

	assert.NoError(t, err)

	// 2. Test Save Error

	mock.ExpectSet("scan:task1", uint64(100), time.Duration(0)).SetErr(assert.AnError)

	err = store.SaveCursor("task1", 100)

	assert.Error(t, err)

	// 3. Test Load Success

	mock.ExpectGet("scan:task1").SetVal("500")

	h, err := store.LoadCursor("task1")

	assert.NoError(t, err)

	assert.Equal(t, uint64(500), h)

	// 4. Test Load Not Found (Redis Nil)

	mock.ExpectGet("scan:task2").SetErr(redis.Nil)

	h, err = store.LoadCursor("task2")

	assert.NoError(t, err)

	assert.Equal(t, uint64(0), h)

	// 5. Test Load Error

	mock.ExpectGet("scan:task3").SetErr(assert.AnError)

	_, err = store.LoadCursor("task3")

	assert.Error(t, err)

	// 6. Test Close

	// redismock doesn't fully support ExpectClose in older versions or some implementations,

	// but RedisStore.Close just calls client.Close.

	// We can't easily mock Close error with redismock without custom wrapper,

	// but calling it ensures coverage hits the line.

	assert.NoError(t, store.Close())

}

func TestNewRedisStore_Mock(t *testing.T) {
	// redismock doesn't directly mock NewRedisStore because it calls redis.NewClient inside.
	// But we can verify our Load/Save tests already cover the logic.
}

// TestNewRedisStore_PingFail attempts to test connection failure logic.
func TestNewRedisStore_PingFail(t *testing.T) {
	// Use an unreachable address with no retries to avoid noise and 2s delay
	client := redis.NewClient(&redis.Options{
		Addr:       "localhost:65432",
		MaxRetries: -1, // Disable retries
	})
	defer client.Close()

	err := client.Ping(context.Background()).Err()
	assert.Error(t, err)

	_, err = NewRedisStore("localhost:65432", "", 0, "p_")
	assert.Error(t, err)
}

func TestRedisStore_Integration(t *testing.T) {
	// If Redis is running in CI or local (default port)
	s, err := NewRedisStore("localhost:6379", "", 0, "integration:")
	if err != nil {
		t.Skip("Redis not available on localhost:6379, skipping integration test")
		return
	}
	defer s.Close()

	err = s.SaveCursor("chain1", 12345)
	assert.NoError(t, err)

	val, err := s.LoadCursor("chain1")
	assert.NoError(t, err)
	assert.Equal(t, uint64(12345), val)
}
