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

func TestFeedbackSubmit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/feedback", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		assert.Equal(t, "bug", body["type"])
		assert.Equal(t, "Ten characters at least here for the message.", body["message"])

		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted"})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"feedback", "submit",
		"--type", "bug",
		"--message", "Ten characters at least here for the message.",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestFeedbackSubmitRejectsInvalidType(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"feedback", "submit",
		"--type", "invalid",
		"--message", "Ten characters at least here for the message.",
		"--api-key", "chr_sk_xxx",
	})
	err := rootCmd.Execute()
	assert.ErrorContains(t, err, "invalid --type")
}

func TestFeedbackSubmitRejectsShortMessage(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"feedback", "submit",
		"--type", "bug",
		"--message", "short",
		"--api-key", "chr_sk_xxx",
	})
	err := rootCmd.Execute()
	assert.ErrorContains(t, err, "at least 10 characters")
}

func TestFeedbackSubmitNoAPIKey(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"feedback", "submit",
		"--type", "bug",
		"--message", "Ten characters at least here for the message.",
	})
	assert.Error(t, rootCmd.Execute())
}
