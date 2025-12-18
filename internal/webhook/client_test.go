package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestWebhookSend(t *testing.T) {
	secret := "my-secret"
	
	// 1. Create Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate Headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("X-Scanner-Signature"))

		// Validate Body
		body, _ := io.ReadAll(r.Body)
		var p Payload
		err := json.Unmarshal(body, &p)
		assert.NoError(t, err)
		assert.Len(t, p.Logs, 1)

		// Validate HMAC signature
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		expectedSig := hex.EncodeToString(h.Sum(nil))
		assert.Equal(t, expectedSig, r.Header.Get("X-Scanner-Signature"))

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// 2. Test Sending
	client := NewClient(Config{URL: ts.URL, Secret: "my-secret"})
	logs := []types.Log{
		{
			Index:   1,
			Address: common.HexToAddress("0x1"),
			Topics:  []common.Hash{common.HexToHash("0x1")},
			Data:    []byte{},
		},
	}
	err := client.Send(context.Background(), logs)
	assert.NoError(t, err)
}

func TestWebhook_Retry(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Set short backoff for faster test
	client := NewClient(Config{
		URL:            ts.URL,
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
	})
	
	logs := []types.Log{{Index: 1}}
	err := client.Send(context.Background(), logs)
	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestWebhook_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewClient(Config{URL: ts.URL, MaxAttempts: 3})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.Send(ctx, []types.Log{{Index: 1}})
	assert.Error(t, err)
}
