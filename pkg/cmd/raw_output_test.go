package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	fn()

	require.NoError(t, w.Close())
	os.Stderr = orig
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(b)
}

const rawEventBody = `{"id":"evt_1","orgId":"org_1","calendarId":"cal_1","title":"Réunion","description":null,"startTime":"2026-11-24T09:00:00.000Z","endTime":"2026-11-24T10:00:00.000Z","status":"confirmed","source":"internal","metadata":{"team":"core","n":3},"deletedAt":null,"createdAt":"2026-11-01T00:00:00.000Z","updatedAt":"2026-11-01T00:00:00.000Z","external_sync":{"status":"failed","reason":"calendar_not_writable"}}`

// -o json --debug must print valid JSON on stdout; the HTTP traces go to
// stderr (#2723).
func TestDebugTracesGoToStderr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[` + rawEventBody + `],"total":1,"limit":50,"offset":0}`))
	}))
	defer srv.Close()

	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			rootCmd := NewRootCmd("test")
			rootCmd.SetArgs([]string{"events", "list", "--calendar", "cal_1", "-o", "json", "--debug", "--api-key", "chr_sk_testSECRET1234", "--base-url", srv.URL})
			require.NoError(t, rootCmd.Execute())
		})
	})

	assert.True(t, json.Valid([]byte(stdout)), "stdout must be valid JSON, got %q", stdout)
	assert.NotContains(t, stdout, ">>>")
	assert.NotContains(t, stdout, "<<<")
	assert.Contains(t, stderr, ">>> GET ")
	assert.Contains(t, stderr, ">>> Authorization: Bearer chr_sk_...1234")
	assert.Contains(t, stderr, "<<< 200")
	assert.NotContains(t, stderr, "SECRET")
}

// -o json / -o yaml print the API response as sent, not a re-marshalled
// client struct that drops fields and reformats values (#2738).
func TestStructuredOutputMirrorsAPIBody(t *testing.T) {
	cases := []struct {
		name string
		args []string
		body string
	}{
		{"events create", []string{"events", "create", "--calendar", "cal_1", "--title", "Réunion", "--start", "2026-11-24T09:00:00Z", "--end", "2026-11-24T10:00:00Z"}, rawEventBody},
		{"events get", []string{"events", "get", "evt_1"}, rawEventBody},
		{"events list-all", []string{"events", "list-all"}, `{"data":[` + rawEventBody + `],"total":1,"limit":50,"offset":0,"next_cursor":null}`},
		{"agents list", []string{"agents", "list"}, `{"data":[{"id":"agt_1","orgId":"org_1","name":"A","status":"active","metadata":{"k":"v"},"working_hours":{"mon":[["09:00","17:00"]]},"createdAt":"2026-11-01T00:00:00.000Z"}],"total":1,"limit":50,"offset":0,"next_cursor":"c_2"}`},
		{"webhooks deliveries", []string{"webhooks", "deliveries", "wh_1"}, `{"data":[{"id":"del_1","event_type":"event.created","status":"failed","attempts":3,"last_response_status":502,"last_error_code":"http_error","last_duration_ms":1234,"created_at":"2026-11-01T00:00:00.000Z"}],"total":1,"limit":50,"offset":0}`},
		{"plans list", []string{"plans", "list"}, `{"data":[{"id":"free","name":"Free","limits":{"agents":1,"calendars":1,"recurring_events":true,"booking_pages":0,"calendar_connections":0}}]}`},
		{"billing", []string{"billing"}, `{"plan":"pro","status":"active","payment_method":{"brand":"visa","last4":"4242"},"current_period_end":"2026-12-01T00:00:00.000Z"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					w.WriteHeader(http.StatusCreated)
				}
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			run := func(format string) string {
				return captureStdout(t, func() {
					rootCmd := NewRootCmd("test")
					rootCmd.SetArgs(append(append([]string{}, tc.args...), "-o", format, "--api-key", "chr_sk_xxx", "--base-url", srv.URL))
					require.NoError(t, rootCmd.Execute())
				})
			}

			// JSON: the same document, every key and value as the API sent it.
			jsonOut := run("json")
			assert.JSONEq(t, tc.body, jsonOut)

			// YAML: the API's keys and values, and JSON objects as mappings
			// (not lists of byte values or lowercased Go field names).
			yamlOut := run("yaml")
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(yamlOut), &node), yamlOut)
			var fromYAML any
			require.NoError(t, node.Decode(&fromYAML))
			roundTrip, err := json.Marshal(fromYAML)
			require.NoError(t, err)
			assert.JSONEq(t, tc.body, string(roundTrip), yamlOut)
		})
	}
}

func TestStructuredYAMLKeepsKeysAndMetadataMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(rawEventBody))
	}))
	defer srv.Close()

	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs([]string{"events", "get", "evt_1", "-o", "yaml", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Contains(t, out, "orgId: org_1")
	assert.Contains(t, out, "external_sync:\n  status: failed")
	assert.Contains(t, out, "metadata:\n  team: core\n  \"n\": 3") // "n" is a YAML 1.1 bool, so the key is quoted
	assert.Contains(t, out, `"2026-11-24T09:00:00.000Z"`)
	assert.NotContains(t, out, "orgid:")
	assert.False(t, strings.Contains(out, "createdat:"))
}

// Locally built data (no API body) still prints YAML with its json keys.
func TestPrintStructuredYAMLUsesJSONKeys(t *testing.T) {
	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("1.2.3")
		rootCmd.SetArgs([]string{"version", "-o", "yaml"})
		require.NoError(t, rootCmd.Execute())
	})
	assert.Equal(t, "version: 1.2.3\n", out)
}
