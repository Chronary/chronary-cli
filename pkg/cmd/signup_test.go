package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthSignupNewOrg(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/agent/sign-up", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		// Public endpoint — must work without an Authorization header.
		assert.Empty(t, r.Header.Get("Authorization"))

		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		assert.Equal(t, "alice@example.com", body["email"])
		assert.Equal(t, "Alice Bot", body["agent_name"])
		assert.Equal(t, "2026-04-17", body["tos_version"])

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"org_id":   "org_abc123",
			"agent_id": "agt_abc123",
			"api_key":  "chr_sk_restricted_abc",
			"message":  "Verification code sent to email",
		})
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	rootCmd := NewRootCmd("test")
	rootCmd.SetOut(out)
	rootCmd.SetArgs([]string{
		"auth", "signup",
		"--email", "alice@example.com",
		"--agent-name", "Alice Bot",
		"--tos-version", "2026-04-17",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestAuthSignupTableShowsNewKeyOnce(t *testing.T) {
	const newKey = "chr_sk_restricted_ONETIMEKEY4321"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"org_id":   "org_abc123",
			"agent_id": "agt_abc123",
			"api_key":  newKey,
			"message":  "Verification code sent to email",
		})
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

	// The new key is the point of signup, so it is shown, but only once: the
	// next-step hint refers to it instead of repeating it.
	assert.Equal(t, 1, strings.Count(out, newKey))
	assert.Contains(t, out, "--api-key <the API key above>")
}

func TestAuthSignupExistingOrgReturnsOnlyMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "Verification code sent to email",
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"auth", "signup",
		"--email", "alice@example.com",
		"--agent-name", "Alice Bot",
		"--tos-version", "2026-04-17",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestAuthSignupRejectsBadEmail(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"auth", "signup",
		"--email", "not-an-email",
		"--agent-name", "Alice Bot",
		"--tos-version", "2026-04-17",
	})
	err := rootCmd.Execute()
	assert.ErrorContains(t, err, "@")
}

func TestAuthVerifySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/agent/verify", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer chr_sk_restricted_abc", r.Header.Get("Authorization"))

		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		assert.Equal(t, "123456", body["otp"])

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"verified": true,
			"message":  "Full access unlocked",
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"auth", "verify",
		"--otp", "123456",
		"--api-key", "chr_sk_restricted_abc",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestAuthVerifyRejectsShortOTP(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"auth", "verify",
		"--otp", "123",
		"--api-key", "chr_sk_restricted_abc",
	})
	err := rootCmd.Execute()
	assert.ErrorContains(t, err, "6 digits")
}

func TestAuthVerifyRequiresAPIKey(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"auth", "verify",
		"--otp", "123456",
	})
	err := rootCmd.Execute()
	assert.ErrorContains(t, err, "no API key")
}
