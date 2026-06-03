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

func newTestScopedKey(id, prefix, agentID string, label *string) client.ScopedAPIKey {
	return client.ScopedAPIKey{
		ID:        id,
		KeyPrefix: prefix,
		AgentID:   agentID,
		Label:     label,
		CreatedAt: time.Now().UTC(),
	}
}

func TestKeysListCommand(t *testing.T) {
	label := "Sales sandbox"
	keys := []client.ScopedAPIKey{
		newTestScopedKey("key_1", "chr_ak_ABCD1234", "agt_1", &label),
		newTestScopedKey("key_2", "chr_ak_WXYZ9876", "agt_2", nil),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/keys", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{"keys": keys})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"keys", "list", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestKeysCreateCommand(t *testing.T) {
	label := "Customer A"
	created := client.CreatedScopedAPIKey{
		ScopedAPIKey: newTestScopedKey("key_new", "chr_ak_ABCD1234", "agt_123", &label),
		Key:          "chr_ak_SECRET123",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/keys", r.URL.Path)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "agt_123", body["agent_id"])
		assert.Equal(t, "Customer A", body["label"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"keys", "create",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--agent", "agt_123",
		"--label", "Customer A",
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestKeysCreateCommandWithFile(t *testing.T) {
	created := client.CreatedScopedAPIKey{
		ScopedAPIKey: newTestScopedKey("key_file", "chr_ak_ABCD1234", "agt_456", nil),
		Key:          "chr_ak_SECRET123",
	}

	input := writeTempJSON(t, "key.json", map[string]any{
		"agent_id": "agt_456",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/keys", r.URL.Path)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "agt_456", body["agent_id"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"keys", "create", input,
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestKeysDeleteCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/keys/key_123", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"keys", "delete", "key_123",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--force",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestKeysCreateRequiresAgentWithoutFile(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"keys", "create",
		"--api-key", "chr_sk_xxx",
	})
	assert.Error(t, rootCmd.Execute())
}

func TestKeysNoAPIKey(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"keys", "list"})
	assert.Error(t, rootCmd.Execute())
}
