package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/usage", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{
			"period_start":         "2026-04-01T00:00:00Z",
			"period_end":           "2026-05-01T00:00:00Z",
			"plan":                 "free",
			"agents":               map[string]int{"used": 2, "limit": 5},
			"calendars":            map[string]int{"used": 3, "limit": 10},
			"events":               map[string]int{"used": 15, "limit": 500},
			"api_calls":            map[string]int{"used": 42, "limit": 1000},
			"webhooks":             map[string]int{"used": 5, "limit": 100},
			"availability_queries": map[string]int{"used": 10, "limit": 200},
			"ical_subscriptions":   map[string]int{"used": 1, "limit": 5},
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"usage", "--api-key", "chr_sk_test", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestUsageNoAPIKey(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"usage"})
	assert.Error(t, rootCmd.Execute())
}
