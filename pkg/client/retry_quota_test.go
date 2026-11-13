package client

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #1936 §1: a monthly plan limit (429 quota_exceeded) is never retried.
func TestClientDoesNotRetryQuotaExceeded(t *testing.T) {
	var attempts int32
	var delays []time.Duration
	originalSleep := retrySleep
	retrySleep = func(d time.Duration) { delays = append(delays, d) }
	defer func() { retrySleep = originalSleep }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Retry-After", "1700000")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"quota_exceeded","message":"Monthly API call limit of 1,000 reached.","metric":"api_calls"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	_, status, err := c.Get("/v1/agents")

	require.Error(t, err)
	assert.Equal(t, http.StatusTooManyRequests, status)
	apiErr, ok := err.(*APIError)
	require.True(t, ok, "expected *APIError, got %T", err)
	assert.Equal(t, "quota_exceeded", apiErr.Type)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
	assert.Empty(t, delays)
}

func TestRetryDelayClampsLongRetryAfter(t *testing.T) {
	assert.Equal(t, 5*time.Second, retryDelay(1, "1700000"))
	assert.Equal(t, 2*time.Second, retryDelay(1, "2"))
}
