package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTermsAccept(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/terms/accept", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		assert.Equal(t, "2026-06-01", body["tos_version"])

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accepted_terms_version": "2026-06-01",
			"accepted_terms_at":      "2026-06-06T16:20:00Z",
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"terms", "accept",
		"--tos-version", "2026-06-01",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestTermsAcceptRequiresTosVersion(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"terms", "accept",
		"--api-key", "chr_sk_xxx",
	})
	assert.Error(t, rootCmd.Execute())
}

func TestTermsAcceptNoAPIKey(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"terms", "accept",
		"--tos-version", "2026-06-01",
	})
	assert.Error(t, rootCmd.Execute())
}
