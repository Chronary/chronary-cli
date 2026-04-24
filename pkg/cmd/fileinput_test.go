package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadBodyFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.json")
	os.WriteFile(path, []byte(`{"name":"Bot","type":"ai"}`), 0o644)

	body, err := readBodyFromFile("@" + path)
	require.NoError(t, err)
	assert.Equal(t, "Bot", body["name"])
	assert.Equal(t, "ai", body["type"])
}

func TestReadBodyFromFileNotAt(t *testing.T) {
	body, err := readBodyFromFile("just-a-string")
	assert.NoError(t, err)
	assert.Nil(t, body)
}

func TestReadBodyFromFileMissing(t *testing.T) {
	_, err := readBodyFromFile("@/nonexistent/file.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading file")
}

func TestReadBodyFromFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte(`not json`), 0o644)

	_, err := readBodyFromFile("@" + path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing JSON")
}

func TestCheckFileArgOutOfBounds(t *testing.T) {
	body, err := checkFileArg([]string{}, 0)
	assert.NoError(t, err)
	assert.Nil(t, body)
}
