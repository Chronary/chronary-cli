package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Chronary/chronary-cli/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAPIKeyPrompt replaces the interactive key prompt for one test.
func stubAPIKeyPrompt(t *testing.T, key string) {
	t.Helper()
	// An exported CHRONARY_BASE_URL on a developer machine must not leak in.
	t.Setenv("CHRONARY_BASE_URL", "")
	orig := promptAPIKey
	promptAPIKey = func() (string, error) { return key, nil }
	t.Cleanup(func() { promptAPIKey = orig })
}

// keyServer answers GET /v1/usage with 200 for goodKey and 401 otherwise, and
// counts every request it receives.
func keyServer(t *testing.T, goodKey string) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		assert.Equal(t, "/v1/usage", r.URL.Path, "login must validate against an authenticated endpoint")
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer "+goodKey {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"type":"authentication_error","message":"Invalid API key","request_id":"req_1"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"plan":"free"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// useLoginDefaultURL points the login default base URL at a test server.
func useLoginDefaultURL(t *testing.T, url string) {
	t.Helper()
	orig := loginDefaultBaseURL
	loginDefaultBaseURL = url
	t.Cleanup(func() { loginDefaultBaseURL = orig })
}

func runLogin(t *testing.T, args ...string) error {
	t.Helper()
	var err error
	captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs(append([]string{"auth", "login"}, args...))
		err = rootCmd.Execute()
	})
	return err
}

// #2559: a key the API rejects is not saved and login fails.
func TestAuthLoginRejectsKeyTheAPIRejects(t *testing.T) {
	useTempConfigDir(t)
	srv, _ := keyServer(t, "chr_sk_good")
	stubAPIKeyPrompt(t, "chr_sk_typo")

	err := runLogin(t, "--base-url", srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rejected this key; nothing was saved: Invalid API key (request_id: req_1)")
	assert.Contains(t, err.Error(), srv.URL, "the error names the URL it validated against")

	cfg, loadErr := config.Load()
	require.NoError(t, loadErr)
	assert.Nil(t, cfg, "nothing may be saved for a rejected key")
}

func TestAuthLoginSavesValidatedKey(t *testing.T) {
	useTempConfigDir(t)
	srv, hits := keyServer(t, "chr_sk_good")
	stubAPIKeyPrompt(t, "chr_sk_good")

	require.NoError(t, runLogin(t, "--base-url", srv.URL, "--profile", "dev"))
	assert.Equal(t, int32(1), atomic.LoadInt32(hits))

	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "dev", cfg.ActiveProfile)
	assert.Equal(t, "chr_sk_good", cfg.Profiles["dev"].APIKey)
	assert.Equal(t, srv.URL, cfg.Profiles["dev"].BaseURL)
}

// #2562: an unparseable config is reported and never overwritten.
func TestAuthLoginRefusesToOverwriteCorruptConfig(t *testing.T) {
	useTempConfigDir(t)
	srv, hits := keyServer(t, "chr_sk_good")
	stubAPIKeyPrompt(t, "chr_sk_good")

	p, err := config.Path()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
	corrupt := []byte(`{"active_profile":"a","profiles":{"a":{"api_key":"chr_sk_a"},"b":{"api_key":"chr_sk_b"},}}`)
	require.NoError(t, os.WriteFile(p, corrupt, 0o600))

	err = runLogin(t, "--base-url", srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be read")
	assert.Equal(t, int32(0), atomic.LoadInt32(hits), "no prompt or request before the config problem is reported")

	after, readErr := os.ReadFile(p)
	require.NoError(t, readErr)
	assert.Equal(t, corrupt, after, "the corrupt config must be left exactly as it was")
}

func TestCorruptConfigIsReportedNotTreatedAsNoKey(t *testing.T) {
	useTempConfigDir(t)
	t.Setenv("CHRONARY_API_KEY", "")
	p, err := config.Path()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
	require.NoError(t, os.WriteFile(p, []byte(`{not json`), 0o600))

	for _, args := range [][]string{
		{"agents", "list"},
		{"auth", "switch", "a"},
		{"auth", "remove", "a"},
		{"auth", "list"},
	} {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs(args)
		var execErr error
		captureStdout(t, func() { execErr = rootCmd.Execute() })
		require.Error(t, execErr, args)
		assert.Contains(t, execErr.Error(), "could not be read", args)
	}
}

func TestCorruptConfigIgnoredWhenAPIKeyFlagGiven(t *testing.T) {
	useTempConfigDir(t)
	p, err := config.Path()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
	require.NoError(t, os.WriteFile(p, []byte(`{not json`), 0o600))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"total":0,"limit":50,"offset":0}`))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"agents", "list", "--api-key", "chr_sk_x", "--base-url", srv.URL, "-o", "json"})
	captureStdout(t, func() { require.NoError(t, rootCmd.Execute()) })
}

// #2527: logging in to a NEW profile while another is active never calls the
// active profile's server and saves no base URL (the default).
func TestAuthLoginNewProfileIgnoresActiveProfileBaseURL(t *testing.T) {
	useTempConfigDir(t)
	devSrv, devHits := keyServer(t, "chr_sk_dev")
	defSrv, defHits := keyServer(t, "chr_sk_prod")
	useLoginDefaultURL(t, defSrv.URL)
	writeTestConfig(t, &config.Config{
		ActiveProfile: "dev",
		Profiles:      map[string]*config.Profile{"dev": {APIKey: "chr_sk_dev", BaseURL: devSrv.URL}},
	})
	stubAPIKeyPrompt(t, "chr_sk_prod")

	require.NoError(t, runLogin(t, "--profile", "prod"))
	assert.Equal(t, int32(0), atomic.LoadInt32(devHits), "the active (dev) profile's URL must not be used")
	assert.Equal(t, int32(1), atomic.LoadInt32(defHits))

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Profiles["prod"].BaseURL, "the default URL is saved as empty")
	assert.Equal(t, "chr_sk_prod", cfg.Profiles["prod"].APIKey)
	assert.Equal(t, devSrv.URL, cfg.Profiles["dev"].BaseURL, "other profiles are untouched")
}

// #2527: a profile already saved with a wrong URL by the old behaviour is
// repaired by re-login: its saved URL is not reused.
func TestAuthLoginRepairsPoisonedProfileURL(t *testing.T) {
	useTempConfigDir(t)
	devSrv, devHits := keyServer(t, "chr_sk_dev")
	defSrv, defHits := keyServer(t, "chr_sk_prod")
	useLoginDefaultURL(t, defSrv.URL)
	writeTestConfig(t, &config.Config{
		ActiveProfile: "dev",
		Profiles: map[string]*config.Profile{
			"dev":  {APIKey: "chr_sk_dev", BaseURL: devSrv.URL},
			"prod": {APIKey: "chr_sk_prod", BaseURL: devSrv.URL},
		},
	})
	stubAPIKeyPrompt(t, "chr_sk_prod")

	require.NoError(t, runLogin(t, "--profile", "prod"))
	assert.Equal(t, int32(0), atomic.LoadInt32(devHits))
	assert.Equal(t, int32(1), atomic.LoadInt32(defHits))
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Profiles["prod"].BaseURL)
}

func TestAuthLoginBaseURLFlagIsSaved(t *testing.T) {
	useTempConfigDir(t)
	defSrv, defHits := keyServer(t, "chr_sk_new")
	flagSrv, flagHits := keyServer(t, "chr_sk_new")
	useLoginDefaultURL(t, defSrv.URL)
	stubAPIKeyPrompt(t, "chr_sk_new")

	require.NoError(t, runLogin(t, "--profile", "staging", "--base-url", flagSrv.URL))
	assert.Equal(t, int32(0), atomic.LoadInt32(defHits))
	assert.Equal(t, int32(1), atomic.LoadInt32(flagHits))
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, flagSrv.URL, cfg.Profiles["staging"].BaseURL)
}

// A failure other than a rejected key says nothing was saved before the
// reason, and names the URL.
func TestAuthLoginServerErrorMessage(t *testing.T) {
	useTempConfigDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	stubAPIKeyPrompt(t, "chr_sk_good")

	err := runLogin(t, "--base-url", srv.URL)
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "key validation against "+srv.URL+" failed; nothing was saved:\n"), err.Error())
	assert.NotContains(t, err.Error(), "..")
	cfg, loadErr := config.Load()
	require.NoError(t, loadErr)
	assert.Nil(t, cfg)
}

// Key-less commands keep working with a corrupt config (#2562 review).
func TestCorruptConfigDoesNotBreakKeylessCommands(t *testing.T) {
	useTempConfigDir(t)
	t.Setenv("CHRONARY_API_KEY", "")
	p, err := config.Path()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
	require.NoError(t, os.WriteFile(p, []byte(`{not json`), 0o600))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","ts":"2026-11-22T00:00:00Z"}`))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"health", "--base-url", srv.URL, "-o", "json"})
	captureStdout(t, func() { require.NoError(t, rootCmd.Execute()) })
}
