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

func newTestAgent(id, name, agentType, status string) client.Agent {
	return client.Agent{
		ID:        id,
		Name:      name,
		Type:      agentType,
		Status:    status,
		Metadata:  json.RawMessage(`{}`),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestAgentsListCommand(t *testing.T) {
	agents := []client.Agent{
		newTestAgent("agt_1", "Agent One", "ai", "active"),
		newTestAgent("agt_2", "Agent Two", "human", "paused"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/agents", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{
			"data":   agents,
			"total":  2,
			"limit":  50,
			"offset": 0,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"agents", "list", "--api-key", "chr_sk_test", "--base-url", srv.URL, "--output", "json"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestAgentsListWithFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "ai", r.URL.Query().Get("type"))
		assert.Equal(t, "active", r.URL.Query().Get("status"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{}, "total": 0, "limit": 10, "offset": 0,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"agents", "list",
		"--api-key", "chr_sk_test",
		"--base-url", srv.URL,
		"--type", "ai",
		"--status", "active",
		"--limit", "10",
		"--output", "json",
	})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestAgentsCreateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Test Bot", body["name"])
		assert.Equal(t, "ai", body["type"])

		w.WriteHeader(201)
		json.NewEncoder(w).Encode(newTestAgent("agt_new", "Test Bot", "ai", "active"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"agents", "create",
		"--api-key", "chr_sk_test",
		"--base-url", srv.URL,
		"--name", "Test Bot",
		"--output", "json",
	})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestAgentsGetCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/agents/agt_123", r.URL.Path)
		json.NewEncoder(w).Encode(newTestAgent("agt_123", "My Agent", "ai", "active"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"agents", "get", "agt_123",
		"--api-key", "chr_sk_test",
		"--base-url", srv.URL,
		"--output", "json",
	})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestAgentsUpdateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Renamed", body["name"])

		json.NewEncoder(w).Encode(newTestAgent("agt_123", "Renamed", "ai", "active"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"agents", "update", "agt_123",
		"--api-key", "chr_sk_test",
		"--base-url", srv.URL,
		"--name", "Renamed",
		"--output", "json",
	})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestAgentsDeleteCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/agents/agt_123", r.URL.Path)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"agents", "delete", "agt_123",
		"--api-key", "chr_sk_test",
		"--base-url", srv.URL,
		"--force",
	})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestAgentsNoAPIKey(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"agents", "list"})
	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestAgentsUpdateNoFlags(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"agents", "update", "agt_123",
		"--api-key", "chr_sk_test",
	})
	err := rootCmd.Execute()
	assert.Error(t, err)
}
