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

func TestWebhook_Error(t *testing.T) {
	// Test HTTP 404 scenario
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := NewClient(Config{URL: ts.URL, Secret: "test-secret"})
	logs := []types.Log{
		{
			Topics: []common.Hash{},
		},
	}
	err := client.Send(context.Background(), logs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}
