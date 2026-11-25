package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Literal snake_case body matching the OpenAPI ICalSubscription schema and
// the API serializer (services/ical-subscriptions.ts toPublic). Deliberately
// not built from client.ICalSubscription, so a casing drift in the struct tags
// fails here instead of round-tripping (#2521).
const apiICalSubJSON = `{"id":"ics_1","agent_id":"agt_1","calendar_id":"cal_1","url":"https://x/f.ics","label":null,"status":"error","last_synced_at":"2026-11-19T10:00:00.000Z","last_error":"HTTP 404","created_at":"2026-11-01T00:00:00.000Z"}`

func icalSnakeServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestICalGetDecodesSnakeCaseJSON(t *testing.T) {
	srv := icalSnakeServer(t, apiICalSubJSON)
	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"ical", "get", "ics_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
		require.NoError(t, rootCmd.Execute())
	})
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &got))
	assert.Equal(t, "agt_1", got["agent_id"])
	assert.Equal(t, "cal_1", got["calendar_id"])
	assert.Equal(t, "HTTP 404", got["last_error"])
	assert.Equal(t, "2026-11-19T10:00:00.000Z", got["last_synced_at"])
	// -o json prints the body as the API sent it, milliseconds included (#2738).
	assert.Equal(t, "2026-11-01T00:00:00.000Z", got["created_at"])
	_, hasCamel := got["agentId"]
	assert.False(t, hasCamel, "no camelCase keys in -o json")
}

func TestICalGetTableShowsSnakeCaseFields(t *testing.T) {
	srv := icalSnakeServer(t, apiICalSubJSON)
	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"ical", "get", "ics_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--no-color"})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Contains(t, out, "agt_1")
	assert.Contains(t, out, "cal_1")
	assert.Contains(t, out, "Last Error:  HTTP 404")
	assert.Contains(t, out, "2026-11-19T10:00:00")
	assert.Contains(t, out, "2026-11-01 00:00:00")
	assert.NotContains(t, out, "0001-01-01")
}

func TestICalListTableShowsCalendarAndLastSync(t *testing.T) {
	srv := icalSnakeServer(t, `{"data":[`+apiICalSubJSON+`],"total":1,"limit":50,"offset":0}`)
	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"ical", "list", "--agent", "agt_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--no-color"})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Contains(t, out, "cal_1")
	assert.Contains(t, out, "2026-11-19T10:00:00")
}
