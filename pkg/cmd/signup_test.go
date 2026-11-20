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

func TestAuthSignupAnonymous(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/agent/sign-up/anonymous", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Empty(t, r.Header.Get("Authorization"))

		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		assert.Equal(t, "Playground", body["agent_name"])
		assert.Equal(t, "2026-04-17", body["tos_version"])
		assert.Equal(t, "agent", body["source"])
		_, hasEmail := body["email"]
		assert.False(t, hasEmail)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"org_id":     "11111111-2222-3333-4444-555555555555",
			"agent_id":   "agt_anon",
			"api_key":    "chr_sk_ANONKEY",
			"claim_url":  "https://console.chronary.ai/claim/anonymous#token=tok",
			"expires_at": "2026-11-20T12:00:00.000Z",
			"plan":       "free-agent-unverified",
		})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{
			"auth", "signup", "--anonymous",
			"--agent-name", "Playground",
			"--tos-version", "2026-04-17",
			"--base-url", srv.URL,
		})
		require.NoError(t, rootCmd.Execute())
	})

	assert.Equal(t, 1, strings.Count(out, "chr_sk_ANONKEY"))
	assert.Contains(t, out, "Claim URL:  https://console.chronary.ai/claim/anonymous#token=tok")
	assert.Contains(t, out, "Expires:    2026-11-20T12:00:00.000Z")
}

func TestAuthSignupAnonymousJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"org_id": "o", "agent_id": "agt_a", "api_key": "chr_sk_X",
			"claim_url": "https://console.chronary.ai/claim/anonymous#token=t", "expires_at": "2026-11-20T12:00:00.000Z", "plan": "free-agent-unverified",
		})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{
			"auth", "signup", "--anonymous", "--agent-name", "P", "--tos-version", "v",
			"--base-url", srv.URL, "--output", "json",
		})
		require.NoError(t, rootCmd.Execute())
	})
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "https://console.chronary.ai/claim/anonymous#token=t", got["claim_url"])
	assert.Equal(t, "chr_sk_X", got["api_key"])
}

func TestAuthSignupAnonymousRejectsEmail(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"auth", "signup", "--anonymous", "--email", "a@b.c",
		"--agent-name", "P", "--tos-version", "v",
	})
	assert.ErrorContains(t, rootCmd.Execute(), "drop --email")
}

func TestAuthSignupAnonymousRequiresTosVersion(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"auth", "signup", "--anonymous", "--agent-name", "P", "--tos-version", ""})
	assert.ErrorContains(t, rootCmd.Execute(), "--tos-version is required")
}

func TestAuthSignupWithoutEmailOrAnonymous(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"auth", "signup", "--agent-name", "P", "--tos-version", "v"})
	assert.ErrorContains(t, rootCmd.Execute(), "--anonymous")
}

func TestAuthSignupAnonymousAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"type": "anonymous_signup_disabled", "message": "Sign-up without an email is not available.", "request_id": "req_1"},
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"auth", "signup", "--anonymous", "--agent-name", "P", "--tos-version", "v",
		"--base-url", srv.URL,
	})
	assert.ErrorContains(t, rootCmd.Execute(), "Sign-up without an email is not available")
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
