package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronary/chronary-cli/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testOrgKey   = "chr_sk_SECRETORGKEYbody1234"
	testAgentKey = "chr_ak_SECRETAGENTKEYbody5678"
)

// useTempConfigDir points os.UserConfigDir at a fresh temp dir on every OS so
// tests never read or write the developer's real CLI config.
func useTempConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)         // Windows
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux
	t.Setenv("HOME", dir)            // macOS
}

func writeTestConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	p, err := config.Path()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
	require.NoError(t, config.SaveTo(p, cfg))
}

func TestAuthStatusMasksKey(t *testing.T) {
	useTempConfigDir(t)

	for _, format := range []string{"table", "json"} {
		t.Run(format, func(t *testing.T) {
			out := captureStdout(t, func() {
				rootCmd := NewRootCmd("test")
				rootCmd.SetArgs([]string{"auth", "status", "--api-key", testOrgKey, "--output", format})
				require.NoError(t, rootCmd.Execute())
			})
			assert.NotContains(t, out, "SECRETORGKEY")
			assert.Contains(t, out, "chr_sk_...1234")
		})
	}
}

func TestAuthListMasksKeys(t *testing.T) {
	useTempConfigDir(t)
	writeTestConfig(t, &config.Config{
		ActiveProfile: "org",
		Profiles: map[string]*config.Profile{
			"org":   {APIKey: testOrgKey},
			"agent": {APIKey: testAgentKey, BaseURL: "https://api-dev.example"},
		},
	})

	t.Run("json", func(t *testing.T) {
		out := captureStdout(t, func() {
			rootCmd := NewRootCmd("test")
			rootCmd.SetArgs([]string{"auth", "list", "--output", "json"})
			require.NoError(t, rootCmd.Execute())
		})
		assert.NotContains(t, out, "SECRET")

		var got struct {
			Active   string                     `json:"active"`
			Profiles map[string]*config.Profile `json:"profiles"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &got))
		assert.Equal(t, "org", got.Active)
		assert.Equal(t, "chr_sk_...1234", got.Profiles["org"].APIKey)
		assert.Equal(t, "chr_ak_...5678", got.Profiles["agent"].APIKey)
		assert.Equal(t, "https://api-dev.example", got.Profiles["agent"].BaseURL)
	})

	t.Run("table", func(t *testing.T) {
		out := captureStdout(t, func() {
			rootCmd := NewRootCmd("test")
			rootCmd.SetArgs([]string{"auth", "list", "--no-color"})
			require.NoError(t, rootCmd.Execute())
		})
		assert.NotContains(t, out, "SECRET")
		assert.Contains(t, out, "chr_sk_...1234")
		assert.Contains(t, out, "chr_ak_...5678")
	})

	// Masking must not touch the saved config.
	p, err := config.Path()
	require.NoError(t, err)
	cfg, err := config.LoadFrom(p)
	require.NoError(t, err)
	assert.Equal(t, testOrgKey, cfg.Profiles["org"].APIKey)
}

func TestMaskedProfilesKeepsNilEntries(t *testing.T) {
	got := maskedProfiles(map[string]*config.Profile{"broken": nil})
	assert.Contains(t, got, "broken")
	assert.Nil(t, got["broken"])
}

func TestDebugOutputMasksAuthorizationHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"health", "--api-key", testOrgKey, "--base-url", srv.URL, "--debug"})
		_ = rootCmd.Execute()
	})
	assert.Contains(t, out, ">>> Authorization: Bearer chr_sk_...1234")
	assert.NotContains(t, out, "SECRETORGKEY")
	assert.Equal(t, 1, strings.Count(out, ">>> Authorization"))
}
