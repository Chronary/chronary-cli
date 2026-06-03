package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWebhook(id, url string, events []string) client.Webhook {
	return client.Webhook{
		ID:        id,
		URL:       url,
		Events:    events,
		Active:    true,
		CreatedAt: time.Now(),
	}
}

func TestWebhooksListCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/webhooks", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{
			"data":  []client.Webhook{newTestWebhook("whk_1", "https://example.com/hook", []string{"agent.created"})},
			"total": 1, "limit": 20, "offset": 0,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "list", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksCreateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "https://example.com/hook", body["url"])
		events := body["events"].([]any)
		assert.Len(t, events, 2)
		w.WriteHeader(201)
		wh := newTestWebhook("whk_new", "https://example.com/hook", []string{"agent.created", "event.created"})
		wh.Secret = "whsec_abc123"
		json.NewEncoder(w).Encode(wh)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"webhooks", "create",
		"--url", "https://example.com/hook",
		"--events", "agent.created,event.created",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksGetCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/webhooks/whk_1", r.URL.Path)
		json.NewEncoder(w).Encode(newTestWebhook("whk_1", "https://example.com/hook", []string{"agent.created"}))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "get", "whk_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksUpdateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, false, body["active"])
		json.NewEncoder(w).Encode(newTestWebhook("whk_1", "https://example.com/hook", []string{"agent.created"}))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "update", "whk_1", "--active=false", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestWebhooksDeleteCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"webhooks", "delete", "whk_1", "--force", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}
