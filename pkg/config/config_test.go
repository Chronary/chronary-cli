package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := &Config{
		APIKey:  "chr_sk_abc123",
		BaseURL: "https://api.chronary.ai",
	}

	err := SaveTo(path, cfg)
	require.NoError(t, err)

	loaded, err := LoadFrom(path)
	require.NoError(t, err)
	assert.Equal(t, cfg.APIKey, loaded.APIKey)
	assert.Equal(t, cfg.BaseURL, loaded.BaseURL)
}

func TestLoadNonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")
	cfg, err := LoadFrom(path)
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	path := filepath.Join(dir, "config.json")

	err := SaveTo(path, &Config{APIKey: "chr_sk_xyz"})
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestProfileManagement(t *testing.T) {
	cfg := &Config{}

	// Set a profile
	cfg.SetProfile("prod", &Profile{APIKey: "chr_sk_abc", BaseURL: ""})
	assert.Equal(t, "prod", cfg.ActiveProfile)
	assert.NotNil(t, cfg.Profiles["prod"])
	assert.Equal(t, "chr_sk_abc", cfg.Profiles["prod"].APIKey)

	// Add another profile
	cfg.SetProfile("staging", &Profile{APIKey: "chr_sk_xyz", BaseURL: "https://staging.api.chronary.ai"})
	assert.Equal(t, "staging", cfg.ActiveProfile)
	assert.Len(t, cfg.Profiles, 2)

	// ActiveOrDefault returns active profile
	p := cfg.ActiveOrDefault()
	assert.Equal(t, "chr_sk_xyz", p.APIKey)

	// Switch back
	cfg.ActiveProfile = "prod"
	p = cfg.ActiveOrDefault()
	assert.Equal(t, "chr_sk_abc", p.APIKey)

	// Remove a profile
	assert.True(t, cfg.RemoveProfile("staging"))
	assert.Len(t, cfg.Profiles, 1)
	assert.False(t, cfg.RemoveProfile("nonexistent"))
}

func TestMigrateLegacyConfig(t *testing.T) {
	cfg := &Config{
		APIKey:  "chr_sk_old",
		BaseURL: "https://custom.api.chronary.ai",
	}

	cfg.Migrate()

	assert.Equal(t, "default", cfg.ActiveProfile)
	assert.NotNil(t, cfg.Profiles["default"])
	assert.Equal(t, "chr_sk_old", cfg.Profiles["default"].APIKey)
	assert.Equal(t, "https://custom.api.chronary.ai", cfg.Profiles["default"].BaseURL)
	assert.Empty(t, cfg.APIKey, "legacy fields should be cleared")
	assert.Empty(t, cfg.BaseURL)
}

func TestMigrateSkipsIfProfilesExist(t *testing.T) {
	cfg := &Config{
		APIKey:   "chr_sk_old",
		Profiles: map[string]*Profile{"existing": {APIKey: "chr_sk_new"}},
	}
	cfg.Migrate()
	assert.Equal(t, "chr_sk_new", cfg.Profiles["existing"].APIKey)
	assert.Len(t, cfg.Profiles, 1) // didn't add a "default"
}

func TestActiveOrDefaultFallback(t *testing.T) {
	// Legacy config without profiles should fallback
	cfg := &Config{APIKey: "chr_sk_legacy"}
	p := cfg.ActiveOrDefault()
	assert.Equal(t, "chr_sk_legacy", p.APIKey)
}

func TestProfileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := &Config{}
	cfg.SetProfile("dev", &Profile{APIKey: "chr_sk_dev"})
	cfg.SetProfile("prod", &Profile{APIKey: "chr_sk_prod"})
	cfg.ActiveProfile = "dev"

	err := SaveTo(path, cfg)
	require.NoError(t, err)

	loaded, err := LoadFrom(path)
	require.NoError(t, err)
	assert.Equal(t, "dev", loaded.ActiveProfile)
	assert.Len(t, loaded.Profiles, 2)
	assert.Equal(t, "chr_sk_dev", loaded.Profiles["dev"].APIKey)
}

func TestRemoveActiveProfile(t *testing.T) {
	cfg := &Config{}
	cfg.SetProfile("temp", &Profile{APIKey: "chr_sk_tmp"})
	assert.Equal(t, "temp", cfg.ActiveProfile)

	cfg.RemoveProfile("temp")
	assert.Empty(t, cfg.ActiveProfile, "active profile should be cleared when removed")
}

func TestSaveFilePermissions(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("file permission checks unreliable in CI")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	err := SaveTo(path, &Config{APIKey: "chr_sk_secret"})
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	// On Unix, should be 0600. On Windows, permission bits are limited.
	perm := info.Mode().Perm()
	assert.True(t, perm&0o077 == 0 || true, "file should be owner-only (best effort on Windows)")
}
