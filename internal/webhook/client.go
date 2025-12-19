package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

// Config holds configuration for the Webhook client.
type Config struct {
	URL            string        `mapstructure:"url"`
	Secret         string        `mapstructure:"secret"`
	MaxAttempts    int           `mapstructure:"max_attempts"`
	InitialBackoff time.Duration `mapstructure:"initial_backoff"`
	MaxBackoff     time.Duration `mapstructure:"max_backoff"`
}

// Client defines the Webhook client
type Client struct {
	cfg        Config
	secret     []byte
	httpClient *http.Client
}

// NewClient initializes a new Webhook client
func NewClient(cfg Config) *Client {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 1 * time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 10 * time.Second
	}

	return &Client{
		cfg:    cfg,
		secret: []byte(cfg.Secret),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Payload defines the data structure sent via webhook to consumers.
type Payload struct {
	Timestamp int64       `json:"timestamp"`
	Logs      []types.Log `json:"logs"`
}

// Send pushes logs with retry logic
func (c *Client) Send(ctx context.Context, logs []types.Log) error {
	if len(logs) == 0 {
		return nil
	}

	payload := Payload{
		Timestamp: time.Now().Unix(),
		Logs:      logs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var lastErr error
	backoff := c.cfg.InitialBackoff

	for i := 0; i < c.cfg.MaxAttempts; i++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if i > 0 {
			// Wait for backoff
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}

			// Exponential backoff
			backoff *= 2
			if backoff > c.cfg.MaxBackoff {
				backoff = c.cfg.MaxBackoff
			}
		}

		err := c.attemptSend(ctx, body)
		if err == nil {
			return nil // Success
		}

		lastErr = err
		// For 4xx client errors (e.g., 400 Bad Request), retries usually don't help.
		// Simplified logic: retry for network errors and 5xx.
		// attemptSend already encapsulates StatusCode check.
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", c.cfg.MaxAttempts, lastErr)
}

func (c *Client) attemptSend(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.URL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "evm-scanner-cli/v1")

	if len(c.secret) > 0 {
		h := hmac.New(sha256.New, c.secret)
		h.Write(body)
		signature := hex.EncodeToString(h.Sum(nil))
		req.Header.Set("X-Scanner-Signature", signature)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Typically only 5xx or 429 should be retried.
		// Simplified: any non-2xx is considered a failure.
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	return nil
}
