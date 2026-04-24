package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeTempJSON writes body as JSON to a temp file and returns the @path arg
// form expected by commands that accept @file inputs.
func writeTempJSON(t *testing.T, name string, body map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	data, err := json.Marshal(body)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return "@" + path
}
