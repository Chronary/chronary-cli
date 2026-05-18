package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/agents", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "chronary-cli/"+Version, r.Header.Get("User-Agent"))
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}, "total": 0, "limit": 50, "offset": 0})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	body, status, err := c.Get("/v1/agents")
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Contains(t, string(body), "data")
}

func TestClientPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Test Agent", body["name"])

		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]string{"id": "agt_123", "name": "Test Agent"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	body, status, err := c.Post("/v1/agents", map[string]string{"name": "Test Agent", "type": "ai"})
	require.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.Contains(t, string(body), "agt_123")
}

func TestClientPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"id": "agt_123", "name": "Updated"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	body, status, err := c.Patch("/v1/agents/agt_123", map[string]string{"name": "Updated"})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Contains(t, string(body), "Updated")
}

func TestClientDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	_, status, err := c.Delete("/v1/agents/agt_123")
	require.NoError(t, err)
	assert.Equal(t, 204, status)
}

func TestClientAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"type":       "not_found",
				"message":    "Agent not found",
				"request_id": "req_abc",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	_, _, err := c.Get("/v1/agents/agt_nope")
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, "not_found", apiErr.Type)
	assert.Equal(t, "req_abc", apiErr.RequestID)
}

func TestClientHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
		w.Write([]byte("Bad Gateway"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	_, _, err := c.Get("/health")
	require.Error(t, err)

	httpErr, ok := err.(*HTTPError)
	require.True(t, ok)
	assert.Equal(t, 502, httpErr.StatusCode)
}

func TestClientRetriesTransientGet(t *testing.T) {
	var attempts int32
	var delays []time.Duration
	originalSleep := retrySleep
	retrySleep = func(d time.Duration) {
		delays = append(delays, d)
	}
	defer func() { retrySleep = originalSleep }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"type":"rate_limited","message":"slow down"}}`))
			return
		}
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}, "total": 0, "limit": 50, "offset": 0})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	body, status, err := c.Get("/v1/agents")

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Contains(t, string(body), "data")
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
	require.Len(t, delays, 1)
	assert.Equal(t, 2*time.Second, delays[0])
}

func TestClientRetriesNetworkGet(t *testing.T) {
	var attempts int32
	originalSleep := retrySleep
	retrySleep = func(time.Duration) {}
	defer func() { retrySleep = originalSleep }()

	c := NewClient("https://example.test", "test-key", false)
	c.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			n := atomic.AddInt32(&attempts, 1)
			if n == 1 {
				return nil, errors.New("temporary connection reset")
			}
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Request:    r,
			}, nil
		}),
	}

	body, status, err := c.Get("/health")

	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, `{"ok":true}`, string(body))
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
}

func TestClientDoesNotRetryMutatingTransientResponse(t *testing.T) {
	var attempts int32
	originalSleep := retrySleep
	retrySleep = func(time.Duration) {}
	defer func() { retrySleep = originalSleep }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", false)
	_, _, err := c.Post("/v1/agents", map[string]string{"name": "Test Agent"})

	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

func TestClientNoAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", false)
	_, status, err := c.Get("/health")
	require.NoError(t, err)
	assert.Equal(t, 200, status)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestRedactDebugBody(t *testing.T) {
	body := []byte(`{"api_key":"chr_sk_SECRET","secret":"whsec_plain","ical_feed_url":"https://api.chronary.ai/ical/tok_secret.ics","ok":true}`)

	redacted := redactDebugBody(body)

	assert.NotContains(t, redacted, "chr_sk_SECRET")
	assert.NotContains(t, redacted, "whsec_plain")
	assert.NotContains(t, redacted, "tok_secret")
	assert.Contains(t, redacted, `"ok":true`)
}
