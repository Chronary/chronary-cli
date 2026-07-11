package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Chronary/chronary-cli/pkg/client"
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
			"webhook_endpoints":    map[string]int{"used": 1, "limit": 3},
			"availability_queries": map[string]int{"used": 10, "limit": 200},
			"ical_subscriptions":   map[string]int{"used": 1, "limit": 5},
			"proposals":            map[string]int{"used": 4, "limit": 500},
			"recurring_events":     map[string]int{"used": 2, "limit": 10},
			"scoped_keys":          map[string]any{"count": 2, "limit": 50},
		})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"usage", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
		require.NoError(t, rootCmd.Execute())
	})

	// The three fields previously missing from UsageResponse must decode and
	// round-trip into the `-o json` output.
	var usage client.UsageResponse
	require.NoError(t, json.Unmarshal([]byte(out), &usage))

	assert.Equal(t, 1, usage.WebhookEndpoints.Used)
	assert.Equal(t, 3, usage.WebhookEndpoints.Limit)
	assert.Equal(t, 4, usage.Proposals.Used)
	assert.Equal(t, 500, usage.Proposals.Limit)
	assert.Equal(t, 2, usage.RecurringEvents.Used)
	assert.Equal(t, 10, usage.RecurringEvents.Limit)
	assert.Equal(t, 2, usage.ScopedKeys.Count)
	require.NotNil(t, usage.ScopedKeys.Limit)
	assert.Equal(t, 50, *usage.ScopedKeys.Limit)

	assert.Contains(t, out, "webhook_endpoints")
	assert.Contains(t, out, "proposals")
	assert.Contains(t, out, "recurring_events")
	assert.Contains(t, out, "scoped_keys")
}

// ScopedKeys.Limit is nullable (unlimited tiers); it must decode as nil.
func TestUsageScopedKeysNullLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"period_start":         "2026-04-01T00:00:00Z",
			"period_end":           "2026-05-01T00:00:00Z",
			"plan":                 "scale",
			"agents":               map[string]int{"used": 0, "limit": 0},
			"calendars":            map[string]int{"used": 0, "limit": 0},
			"events":               map[string]int{"used": 0, "limit": 0},
			"api_calls":            map[string]int{"used": 0, "limit": 0},
			"webhooks":             map[string]int{"used": 0, "limit": 0},
			"webhook_endpoints":    map[string]int{"used": 0, "limit": 0},
			"availability_queries": map[string]int{"used": 0, "limit": 0},
			"ical_subscriptions":   map[string]int{"used": 0, "limit": 0},
			"proposals":            map[string]int{"used": 0, "limit": 0},
			"scoped_keys":          map[string]any{"count": 7, "limit": nil},
		})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"usage", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
		require.NoError(t, rootCmd.Execute())
	})

	var usage client.UsageResponse
	require.NoError(t, json.Unmarshal([]byte(out), &usage))
	assert.Equal(t, 7, usage.ScopedKeys.Count)
	assert.Nil(t, usage.ScopedKeys.Limit)
}

func TestUsageNoAPIKey(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"usage"})
	assert.Error(t, rootCmd.Execute())
}

// captureStdout redirects os.Stdout for the duration of fn and returns whatever
// was written. Used to assert on commands that print structured output via
// output.PrintJSON (which writes directly to os.Stdout).
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = orig
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(b)
}
