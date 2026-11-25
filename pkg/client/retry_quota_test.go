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

// #2558: a 429/503 whose Retry-After is longer than 60 s cannot succeed inside
// the retry window, so it is returned at once instead of retried.
func TestClientDoesNotRetryRetryAfterBeyondCap(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		ra     string
		body   string
	}{
		{"ical cooldown 429", http.StatusTooManyRequests, "900", `{"error":{"type":"rate_limit_error","message":"A sync for this feed started recently."}}`},
		{"daily cap 503", http.StatusServiceUnavailable, "36000", `{"error":{"type":"anonymous_signup_capacity","message":"Sign-up capacity reached for today."}}`},
		{"huge delta-seconds", http.StatusTooManyRequests, "99999999999", `{"error":{"type":"rate_limit_error","message":"Too many requests"}}`},
		// 9.3e9 s is past time.Duration's range and wraps negative when multiplied.
		{"wrapping delta-seconds", http.StatusTooManyRequests, "9300000000", `{"error":{"type":"rate_limit_error","message":"Too many requests"}}`},
		{"http date", http.StatusTooManyRequests, time.Now().Add(2 * time.Hour).UTC().Format(http.TimeFormat), `{"error":{"type":"rate_limit_error","message":"Too many requests"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var attempts int32
			var delays []time.Duration
			originalSleep := retrySleep
			retrySleep = func(d time.Duration) { delays = append(delays, d) }
			defer func() { retrySleep = originalSleep }()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attempts, 1)
				w.Header().Set("Retry-After", tc.ra)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			c := NewClient(srv.URL, "test-key", false)
			_, status, err := c.Get("/v1/agents")

			require.Error(t, err)
			assert.Equal(t, tc.status, status)
			_, ok := err.(*APIError)
			require.True(t, ok, "expected *APIError, got %T", err)
			assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
			assert.Empty(t, delays)
		})
	}
}

func TestClientRetriesRetryAfterWithinCap(t *testing.T) {
	var attempts int32
	var delays []time.Duration
	originalSleep := retrySleep
	retrySleep = func(d time.Duration) { delays = append(delays, d) }
	defer func() { retrySleep = originalSleep }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"type":"temporarily_disabled","message":"busy"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[],"total":0,"limit":20,"offset":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	_, status, err := c.Get("/v1/agents")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
	// A Retry-After within the cap keeps the CLI's 5 s per-wait cap.
	assert.Equal(t, []time.Duration{5 * time.Second}, delays)
}

func TestRetryDelayDoesNotOverflowOnHugeRetryAfter(t *testing.T) {
	assert.Equal(t, 5*time.Second, retryDelay(1, "99999999999"))
	assert.Equal(t, 5*time.Second, retryDelay(1, "9300000000"))
}
