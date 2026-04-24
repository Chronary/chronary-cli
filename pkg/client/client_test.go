package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/agents", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "chronary-cli", r.Header.Get("User-Agent"))
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
