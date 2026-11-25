package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
			"agents":               map[string]int{"used": 1, "limit": 1},
			"calendars":            map[string]int{"used": 1, "limit": 1},
			"events":               map[string]int{"used": 15, "limit": 100},
			"api_calls":            map[string]int{"used": 42, "limit": 1000},
			"webhooks":             map[string]int{"used": 5, "limit": 1000},
			"webhook_endpoints":    map[string]int{"used": 1, "limit": 1},
			"availability_queries": map[string]int{"used": 10, "limit": 100},
			"ical_subscriptions":   map[string]int{"used": 0, "limit": 0},
			"proposals":            map[string]int{"used": 0, "limit": 0},
			"recurring_events":     map[string]int{"used": 1, "limit": 1},
			"calendar_connections": map[string]int{"used": 1, "limit": 5},
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
	require.NotNil(t, usage.WebhookEndpoints.Limit)
	assert.Equal(t, 1, *usage.WebhookEndpoints.Limit)
	assert.Equal(t, 0, usage.Proposals.Used)
	require.NotNil(t, usage.Proposals.Limit)
	assert.Equal(t, 0, *usage.Proposals.Limit)
	assert.Equal(t, 1, usage.RecurringEvents.Used)
	require.NotNil(t, usage.RecurringEvents.Limit)
	assert.Equal(t, 1, *usage.RecurringEvents.Limit)
	assert.Equal(t, 1, usage.CalendarConnections.Used)
	require.NotNil(t, usage.CalendarConnections.Limit)
	assert.Equal(t, 5, *usage.CalendarConnections.Limit)
	assert.Equal(t, 2, usage.ScopedKeys.Count)
	require.NotNil(t, usage.ScopedKeys.Limit)
	assert.Equal(t, 50, *usage.ScopedKeys.Limit)

	assert.Contains(t, out, "webhook_endpoints")
	assert.Contains(t, out, "proposals")
	assert.Contains(t, out, "recurring_events")
	assert.Contains(t, out, "calendar_connections")
	assert.Contains(t, out, "scoped_keys")
}

// ScopedKeys.Limit and every UsageCounter.Limit are nullable (unlimited
// tiers); they must decode as nil and stay null in -o json (#2526).
func TestUsageScopedKeysNullLimit(t *testing.T) {
	unlimited := map[string]any{"used": 0, "limit": nil}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"period_start":         "2026-04-01T00:00:00Z",
			"period_end":           "2026-05-01T00:00:00Z",
			"plan":                 "scale",
			"agents":               unlimited,
			"calendars":            unlimited,
			"events":               unlimited,
			"api_calls":            unlimited,
			"webhooks":             unlimited,
			"webhook_endpoints":    unlimited,
			"availability_queries": unlimited,
			"ical_subscriptions":   unlimited,
			"proposals":            unlimited,
			"booking_pages":        map[string]any{"used": 3, "limit": nil},
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
	assert.Nil(t, usage.BookingPages.Limit)
	assert.Equal(t, 3, usage.BookingPages.Used)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(out), &raw))
	var bookingPages map[string]any
	require.NoError(t, json.Unmarshal(raw["booking_pages"], &bookingPages))
	limit, present := bookingPages["limit"]
	assert.True(t, present, "booking_pages.limit must be present")
	assert.Nil(t, limit, "null (unlimited) must stay null in -o json, not become 0")
}

// A limit of 0 (feature not in the plan) must not render as "unlimited";
// only a null limit is unlimited (#2526).
func TestUsageRowZeroVersusNullLimit(t *testing.T) {
	zero, hundred := 0, 100
	assert.Equal(t, []string{"Proposals", "0", "0", "0"},
		usageRow("Proposals", client.UsageCounter{Used: 0, Limit: &zero}, true))
	assert.Equal(t, []string{"Booking Pages", "3", "unlimited", "-"},
		usageRow("Booking Pages", client.UsageCounter{Used: 3, Limit: nil}, true))
	assert.Equal(t, []string{"Events", "15", "100", "85"},
		usageRow("Events", client.UsageCounter{Used: 15, Limit: &hundred}, true))
}

func TestUsageTableFreePlanShowsZeroNotUnlimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"period_start":       "2026-11-01T00:00:00Z",
			"period_end":         "2026-12-01T00:00:00Z",
			"plan":               "free",
			"events":             map[string]any{"used": 1, "limit": 100},
			"ical_subscriptions": map[string]any{"used": 0, "limit": 0},
			"proposals":          map[string]any{"used": 0, "limit": 0},
		})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"usage", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--no-color"})
		require.NoError(t, rootCmd.Execute())
	})
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "iCal Subscriptions") || strings.Contains(line, "Proposals") {
			assert.NotContains(t, line, "unlimited", line)
		}
	}
}

func TestUsageHistoryCommand(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/usage/history", r.URL.Path)
		gotQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode(map[string]any{
			"plan":   "free",
			"months": 2,
			"limits": map[string]any{"api_calls": 1000, "events": 100, "webhook_deliveries": 1000, "availability_queries": 100, "proposals": nil},
			"data": []map[string]any{
				{"period": "2026-09", "period_start": "2026-09-01T00:00:00.000Z", "period_end": "2026-09-30T23:59:59.000Z", "is_current": false, "metrics": map[string]int{"api_calls": 41220}},
				{"period": "2026-10", "period_start": "2026-10-01T00:00:00.000Z", "period_end": "2026-10-31T23:59:59.000Z", "is_current": true, "metrics": map[string]int{"api_calls": 12480}},
			},
		})
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"usage", "history", "--months", "2", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Equal(t, "months=2", gotQuery)

	var history client.UsageHistoryResponse
	require.NoError(t, json.Unmarshal([]byte(out), &history))
	require.Len(t, history.Data, 2)
	assert.True(t, history.Data[1].IsCurrent)
	assert.Equal(t, 41220, history.Data[0].Metrics.APICalls)
	assert.Nil(t, history.Limits.Proposals)

	// Table output labels the current month.
	table := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"usage", "history", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--no-color"})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Equal(t, "months=6", gotQuery)
	assert.Contains(t, table, "2026-10 (to date)")
	assert.Contains(t, table, "41220")
}

func TestUsageHistoryRejectsBadMonths(t *testing.T) {
	for _, m := range []string{"0", "13"} {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"usage", "history", "--months", m, "--api-key", "chr_sk_xxx", "--base-url", "http://127.0.0.1:1"})
		assert.Error(t, rootCmd.Execute(), "months=%s", m)
	}
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
