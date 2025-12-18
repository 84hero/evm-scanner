package sink

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/84hero/evm-scanner/internal/webhook"
	"github.com/84hero/evm-scanner/pkg/decoder"
	"github.com/IBM/sarama"
	"github.com/ethereum/go-ethereum/core/types"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

// DecodedLog wraps raw log and its decoded result
type DecodedLog struct {
	Log         types.Log           `json:"log"`
	DecodedData *decoder.DecodedLog `json:"decoded,omitempty"`
	EventName   string              `json:"event_name,omitempty"`
}

// Output defines the interface for event output pipeline
type Output interface {
	Name() string
	Send(ctx context.Context, logs []DecodedLog) error
	Close() error
}

// --- 1. Webhook Output ---

type WebhookOutput struct {
	client   *webhook.Client
	async    bool
	queue    chan []types.Log
	wg       sync.WaitGroup
	closed   bool
	closedMu sync.Mutex
}

type WebhookConfig struct {
	URL            string
	Secret         string
	MaxAttempts    int
	InitialBackoff string
	MaxBackoff     string
	Async          bool
	BufferSize     int
	Workers        int
}

func NewWebhookOutput(url, secret string, maxAttempts int, initialBackoff, maxBackoff string, async bool, bufferSize, workers int) *WebhookOutput {
	// Duration conversion logic is handled at the application layer.
	// We receive basic types or a config here.
	client := webhook.NewClient(webhook.Config{
		URL:         url,
		Secret:      secret,
		MaxAttempts: maxAttempts,
		// internal webhook library currently receives time.Duration directly.
		// Assuming app layer handles this or we tweak internal library.
	})

	wo := &WebhookOutput{
		client: client,
		async:  async,
	}

	if async {
		if bufferSize <= 0 {
			bufferSize = 1000
		}
		if workers <= 0 {
			workers = 1
		}
		wo.queue = make(chan []types.Log, bufferSize)
		for i := 0; i < workers; i++ {
			wo.wg.Add(1)
			go wo.worker()
		}
	}

	return wo
}

func (w *WebhookOutput) Name() string { return "webhook" }

func (w *WebhookOutput) worker() {
	defer w.wg.Done()
	for logs := range w.queue {
		if err := w.client.Send(context.Background(), logs); err != nil {
			fmt.Fprintf(os.Stderr, "[Webhook Async Error] %v\n", err)
		}
	}
}

func (w *WebhookOutput) Send(ctx context.Context, logs []DecodedLog) error {
	var rawLogs []types.Log
	for _, l := range logs {
		rawLogs = append(rawLogs, l.Log)
	}

	if w.async {
		w.closedMu.Lock()
		defer w.closedMu.Unlock()
		if w.closed {
			return fmt.Errorf("webhook output is closed")
		}
		select {
		case w.queue <- rawLogs:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return w.client.Send(ctx, rawLogs)
}

func (w *WebhookOutput) Close() error {
	if w.async {
		w.closedMu.Lock()
		if !w.closed {
			w.closed = true
			close(w.queue)
		}
		w.closedMu.Unlock()
		w.wg.Wait()
	}
	return nil
}

// --- 2. File Output ---

type FileOutput struct {
	path string
	mu   sync.Mutex
	file *os.File
}

func NewFileOutput(path string) (*FileOutput, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileOutput{path: path, file: f}, nil
}

func (f *FileOutput) Name() string { return "file" }

func (f *FileOutput) Send(ctx context.Context, logs []DecodedLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	enc := json.NewEncoder(f.file)
	for _, l := range logs {
		if err := enc.Encode(l); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileOutput) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

// --- 3. Console Output ---

type ConsoleOutput struct {
	mu sync.Mutex
}

func NewConsoleOutput() *ConsoleOutput {
	return &ConsoleOutput{}
}

func (c *ConsoleOutput) Name() string { return "console" }

func (c *ConsoleOutput) Send(ctx context.Context, logs []DecodedLog) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	enc := json.NewEncoder(os.Stdout)
	for _, l := range logs {
		if err := enc.Encode(l); err != nil {
			return err
		}
	}
	return nil
}

func (c *ConsoleOutput) Close() error { return nil }

// --- 4. PostgreSQL Output ---

type PostgresOutput struct {
	db    *sql.DB
	table string
}

func NewPostgresOutput(url, table string) (*PostgresOutput, error) {
	if match, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", table); !match {
		return nil, fmt.Errorf("invalid table name: %s", table)
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			block_number BIGINT,
			tx_hash TEXT,
			log_index INT,
			event_name TEXT,
			data JSONB,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE (tx_hash, log_index)
		);
		CREATE INDEX IF NOT EXISTS idx_%s_block ON %s (block_number);
	`, table, table, table)
	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}
	return &PostgresOutput{db: db, table: table}, nil
}

func (p *PostgresOutput) Name() string { return "postgres" }

func (p *PostgresOutput) Send(ctx context.Context, logs []DecodedLog) error {
	if len(logs) == 0 {
		return nil
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	valueStrings := make([]string, 0, len(logs))
	valueArgs := make([]interface{}, 0, len(logs)*5)
	for i, l := range logs {
		jsonData, _ := json.Marshal(l)
		n := i * 5
		valueStrings = append(valueStrings, fmt.Sprintf("($%%d, $%%d, $%%d, $%%d, $%%d)", n+1, n+2, n+3, n+4, n+5))
		valueArgs = append(valueArgs, l.Log.BlockNumber, l.Log.TxHash.Hex(), l.Log.Index, l.EventName, jsonData)
	}
	stmt := fmt.Sprintf("INSERT INTO %s (block_number, tx_hash, log_index, event_name, data) VALUES %s ON CONFLICT (tx_hash, log_index) DO NOTHING", p.table, strings.Join(valueStrings, ","))
	_, err = tx.ExecContext(ctx, stmt, valueArgs...)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (p *PostgresOutput) Close() error { return p.db.Close() }

// --- 5. Redis Output ---

type RedisOutput struct {
	client *redis.Client
	key    string
	mode   string
}

func NewRedisOutput(addr, password string, db int, key, mode string) (*RedisOutput, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &RedisOutput{client: rdb, key: key, mode: mode}, nil
}

func (r *RedisOutput) Name() string { return "redis" }

func (r *RedisOutput) Send(ctx context.Context, logs []DecodedLog) error {
	pipe := r.client.Pipeline()
	for _, l := range logs {
		data, _ := json.Marshal(l)
		if r.mode == "pubsub" {
			pipe.Publish(ctx, r.key, data)
		} else {
			pipe.LPush(ctx, r.key, data)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisOutput) Close() error { return r.client.Close() }

// --- 6. Kafka Output ---

type KafkaOutput struct {
	producer sarama.SyncProducer
	topic    string
}

func NewKafkaOutput(brokers []string, topic, user, password string) (*KafkaOutput, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	if user != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = user
		config.Net.SASL.Password = password
	}
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &KafkaOutput{producer: producer, topic: topic}, nil
}

func (k *KafkaOutput) Name() string { return "kafka" }

func (k *KafkaOutput) Send(ctx context.Context, logs []DecodedLog) error {
	var msgs []*sarama.ProducerMessage
	for _, l := range logs {
		data, _ := json.Marshal(l)
		msgs = append(msgs, &sarama.ProducerMessage{
			Topic: k.topic,
			Key:   sarama.StringEncoder(l.Log.TxHash.Hex()),
			Value: sarama.ByteEncoder(data),
		})
	}
	return k.producer.SendMessages(msgs)
}

func (k *KafkaOutput) Close() error { return k.producer.Close() }

// --- 7. RabbitMQ Output ---

type RabbitMQOutput struct {
	conn       *amqp.Connection
	ch         *amqp.Channel
	exchange   string
	routingKey string
}

func NewRabbitMQOutput(url, exchange, routingKey, queueName string, durable bool) (*RabbitMQOutput, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if exchange != "" {
		err = ch.ExchangeDeclare(exchange, "topic", durable, false, false, false, nil)
		if err != nil {
			ch.Close(); conn.Close(); return nil, err
		}
	}
	if queueName != "" {
		q, _ := ch.QueueDeclare(queueName, durable, false, false, false, nil)
		ch.QueueBind(q.Name, routingKey, exchange, false, nil)
	}
	return &RabbitMQOutput{conn: conn, ch: ch, exchange: exchange, routingKey: routingKey}, nil
}

func (r *RabbitMQOutput) Name() string { return "rabbitmq" }

func (r *RabbitMQOutput) Send(ctx context.Context, logs []DecodedLog) error {
	for _, l := range logs {
		data, _ := json.Marshal(l)
		err := r.ch.PublishWithContext(ctx, r.exchange, r.routingKey, false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         data,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RabbitMQOutput) Close() error {
	r.ch.Close()
	return r.conn.Close()
}