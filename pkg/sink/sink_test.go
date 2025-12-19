package sink

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestFileOutput(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "events_*.jsonl")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	fo, err := NewFileOutput(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "file", fo.Name())

	logs := []DecodedLog{{Log: types.Log{Index: 1, Topics: []common.Hash{}}}}
	err = fo.Send(context.Background(), logs)
	assert.NoError(t, err)

	err = fo.Close()
	assert.NoError(t, err)

	// Verify content
	data, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	var decoded DecodedLog
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), decoded.Log.Index)
}

func TestFileOutput_Fail(t *testing.T) {
	// Try to open a directory as a file
	_, err := NewFileOutput("/")
	assert.Error(t, err)
}

func TestRedisOutput(t *testing.T) {
	db, mock := redismock.NewClientMock()
	ro := &RedisOutput{
		client: db,
		key:    "test_key",
		mode:   "list",
	}
	assert.Equal(t, "redis", ro.Name())

	logs := []DecodedLog{{Log: types.Log{Index: 1}}}
	data, _ := json.Marshal(logs[0])

	mock.ExpectLPush("test_key", data).SetVal(1)
	err := ro.Send(context.Background(), logs)
	assert.NoError(t, err)

	// Test PubSub mode
	ro.mode = "pubsub"
	mock.ExpectPublish("test_key", data).SetVal(1)
	err = ro.Send(context.Background(), logs)
	assert.NoError(t, err)

	err = ro.Close()
	assert.NoError(t, err)
}

func TestWebhookOutput_Sync(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	wo := NewWebhookOutput(ts.URL, "secret", 1, "1s", "10s", false, 0, 0)
	logs := []DecodedLog{{Log: types.Log{Index: 1}}}
	err := wo.Send(context.Background(), logs)
	assert.NoError(t, err)
}

func TestKafkaOutput_Init(t *testing.T) {
	ko, err := NewKafkaOutput([]string{"localhost:9092"}, "test", "", "")
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, ko)
		ko.Close()
	}
}

func TestRabbitMQOutput_Init(t *testing.T) {
	ro, err := NewRabbitMQOutput("amqp://guest:guest@localhost:5672/", "ex", "key", "q", true)
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, ro)
		ro.Close()
	}
}

func TestRedisOutput_Init(t *testing.T) {
	ro, err := NewRedisOutput("localhost:65432", "", 0, "key", "list")
	assert.Error(t, err)
	assert.Nil(t, ro)
}

func TestPostgresOutput_Init(t *testing.T) {
	po, err := NewPostgresOutput("invalid", "table")
	assert.Error(t, err)
	assert.Nil(t, po)
}

func TestPostgresOutput_Send_Multiple(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	p := &PostgresOutput{db: db, table: "events"}

	logs := []DecodedLog{
		{Log: types.Log{BlockNumber: 100, TxHash: common.HexToHash("0x1"), Index: 1}, EventName: "E1"},
		{Log: types.Log{BlockNumber: 101, TxHash: common.HexToHash("0x2"), Index: 2}, EventName: "E2"},
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO events").
		WithArgs(
			uint64(100), "0x0000000000000000000000000000000000000000000000000000000000000001", uint(1), "E1", sqlmock.AnyArg(),
			uint64(101), "0x0000000000000000000000000000000000000000000000000000000000000002", uint(2), "E2", sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(2, 2))
	mock.ExpectCommit()

	err := p.Send(context.Background(), logs)
	assert.NoError(t, err)
}

func TestPostgresOutput_Send_Empty(t *testing.T) {
	p := &PostgresOutput{}
	err := p.Send(context.Background(), []DecodedLog{})
	assert.NoError(t, err)
}

func TestSink_InterfaceCompliance(t *testing.T) {
	sinks := []struct {
		name string
		s    Output
	}{
		{"console", NewConsoleOutput()},
		{"webhook", &WebhookOutput{}},
		{"file", &FileOutput{}},
		{"postgres", &PostgresOutput{}},
		{"redis", &RedisOutput{}},
		{"kafka", &KafkaOutput{}},
		{"rabbitmq", &RabbitMQOutput{}},
	}

	for _, tt := range sinks {
		assert.Equal(t, tt.name, tt.s.Name())
	}
}

func TestPostgresOutput_Close(t *testing.T) {
	db, mock, _ := sqlmock.New()
	p := &PostgresOutput{db: db}
	mock.ExpectClose()
	assert.NoError(t, p.Close())
}

func TestPostgresOutput_Safety(t *testing.T) {
	_, err := NewPostgresOutput("postgres://localhost", "valid_table")
	assert.NotContains(t, err.Error(), "invalid table name")

	_, err = NewPostgresOutput("postgres://localhost", "events; DROP TABLE users;")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid table name")
}

func TestPostgresOutput_Send(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	p := &PostgresOutput{db: db, table: "events"}

	logs := []DecodedLog{
		{
			Log:       types.Log{BlockNumber: 100, TxHash: common.HexToHash("0xabc"), Index: 1},
			EventName: "Transfer",
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO events").
		WithArgs(uint64(100), "0x0000000000000000000000000000000000000000000000000000000000000abc", uint(1), "Transfer", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := p.Send(context.Background(), logs)
	assert.NoError(t, err)
}

func TestWebhookOutput_Async(t *testing.T) {
	called := make(chan bool, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called <- true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	wo := NewWebhookOutput(ts.URL, "secret", 1, "1s", "10s", true, 10, 1)
	logs := []DecodedLog{{Log: types.Log{Index: 1}}}

	start := time.Now()
	err := wo.Send(context.Background(), logs)
	assert.NoError(t, err)
	// Must return immediately in async mode
	assert.Less(t, time.Since(start), 100*time.Millisecond)

	// Wait for background worker to deliver
	select {
	case <-called:
	case <-time.After(1 * time.Second):
		t.Fatal("Async webhook was never delivered")
	}

	err = wo.Close()
	assert.NoError(t, err)
}

func TestConsoleOutput(t *testing.T) {
	c := NewConsoleOutput()
	assert.Equal(t, "console", c.Name())
	err := c.Send(context.Background(), []DecodedLog{{Log: types.Log{Index: 1, Topics: []common.Hash{}}}})
	assert.NoError(t, err)
	assert.NoError(t, c.Close())
}
