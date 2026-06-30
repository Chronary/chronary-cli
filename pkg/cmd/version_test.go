package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCommandJSON(t *testing.T) {
	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("1.2.3")
		rootCmd.SetArgs([]string{"version", "--output", "json"})
		require.NoError(t, rootCmd.Execute())
	})

	var parsed struct {
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "version -o json must emit valid JSON")
	assert.Equal(t, "1.2.3", parsed.Version)
}

func TestVersionCommandDefault(t *testing.T) {
	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("1.2.3")
		rootCmd.SetArgs([]string{"version"})
		require.NoError(t, rootCmd.Execute())
	})

	assert.Equal(t, "chronary 1.2.3", strings.TrimSpace(out))
	assert.False(t, strings.HasPrefix(strings.TrimSpace(out), "{"), "default mode must be plain text, not JSON")
}
