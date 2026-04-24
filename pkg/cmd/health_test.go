package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"ts":     "2026-04-11T00:00:00.000Z",
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"health", "--base-url", srv.URL, "--output", "json"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestHealthCommandFailure(t *testing.T) {
	// Point at a server that returns 502
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
		w.Write([]byte("Bad Gateway"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"health", "--base-url", srv.URL})
	err := rootCmd.Execute()
	require.Error(t, err)
}
