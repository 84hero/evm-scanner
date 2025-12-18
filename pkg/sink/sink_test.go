package sink

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestPostgresOutput_Safety(t *testing.T) {
	// 1. Valid table name
	_, err := NewPostgresOutput("postgres://localhost", "valid_table")
	// Expected error because DB is not reachable, but check if it passed regex
	assert.NotContains(t, err.Error(), "invalid table name")

	// 2. Invalid table name (SQL Injection attempt)
	_, err = NewPostgresOutput("postgres://localhost", "events; DROP TABLE users;")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid table name")
}

func TestPostgresOutput_Send(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	p := &PostgresOutput{
		db:    db,
		table: "events",
	}

	logs := []DecodedLog{
		{
			Log: types.Log{
				BlockNumber: 100,
				TxHash:      common.HexToHash("0xabc"),
				Index:       1,
			},
			EventName: "Transfer",
		},
	}

	// Expect Batch Insert SQL
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO events").
		WithArgs(uint64(100), "0x0000000000000000000000000000000000000000000000000000000000000abc", uint(1), "Transfer", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := p.Send(context.Background(), logs)
	assert.NoError(t, err)
}

func TestWebhookOutput_Async(t *testing.T) {
	// Note: Testing actual HTTP calls would require httptest, 
	// here we test the queueing and graceful shutdown.
	
	wo := NewWebhookOutput("http://localhost", "secret", 1, "1s", "10s", true, 10, 1)
	
	logs := []DecodedLog{{Log: types.Log{Index: 1}}}
	
	// Test Send (should return immediately)
	start := time.Now()
	err := wo.Send(context.Background(), logs)
	assert.NoError(t, err)
	assert.Less(t, time.Since(start), 50*time.Millisecond)

	// Test Close (should wait for workers)
	err = wo.Close()
	assert.NoError(t, err)
}

func TestConsoleOutput(t *testing.T) {
	c := NewConsoleOutput()
	assert.Equal(t, "console", c.Name())
	err := c.Send(context.Background(), []DecodedLog{{Log: types.Log{Index: 1}}})
	assert.NoError(t, err)
}
