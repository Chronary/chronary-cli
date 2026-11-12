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

const waitlistMessage = "Thanks, you're on the list. We'll email you when your spot opens."

func TestAuthWaitlistPrintsMessageOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/waitlist", r.URL.Path)
		assert.Empty(t, r.Header.Get("Authorization"))
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		assert.Equal(t, "dev@acme.ai", body["email"])
		assert.Equal(t, "An agent", body["intent"])

		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"waitlisted": true, "message": waitlistMessage})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{
			"auth", "waitlist",
			"--email", "dev@acme.ai",
			"--intent", "An agent",
			"--base-url", srv.URL,
		})
		require.NoError(t, rootCmd.Execute())
	})

	assert.Contains(t, out, waitlistMessage)
	assert.NotContains(t, out, "Org ID")
}

func TestAuthSignupWaitlistedPrintsMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"waitlisted": true, "message": "Sign-ups are paused."})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{
			"auth", "signup",
			"--email", "alice@example.com",
			"--agent-name", "Alice Bot",
			"--tos-version", "2026-04-17",
			"--base-url", srv.URL,
		})
		require.NoError(t, rootCmd.Execute())
	})

	assert.Contains(t, out, "Sign-ups are paused.")
	assert.NotContains(t, out, "No new credentials")
}
