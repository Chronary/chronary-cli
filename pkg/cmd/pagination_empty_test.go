package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allListCommands is every `--all` command with the page size it asks for.
var allListCommands = []struct {
	name     string
	args     []string
	pageSize int
}{
	{"agents", []string{"agents", "list"}, pageSizeAgents},
	{"calendars", []string{"calendars", "list"}, pageSizeCalendars},
	{"calendar events", []string{"events", "list", "--calendar", "cal_1"}, pageSizeEvents},
	{"org events", []string{"events", "list-all"}, pageSizeOrgEvents},
	{"booking pages", []string{"booking-pages", "list"}, pageSizeBookingPages},
	{"ical subscriptions", []string{"ical", "list", "--agent", "agt_1"}, pageSizeICalSubscriptions},
	{"proposals", []string{"scheduling", "list"}, pageSizeProposals},
	{"webhooks", []string{"webhooks", "list"}, pageSizeWebhooks},
}

type listEnvelope struct {
	Data   []json.RawMessage `json:"data"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

// runAllJSON runs `<args> --all --output json` against srv and returns stdout.
func runAllJSON(t *testing.T, srv *httptest.Server, args []string, extra ...string) string {
	t.Helper()
	rootCmd := NewRootCmd("test")
	full := append(append([]string{}, args...), "--all", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json")
	rootCmd.SetArgs(append(full, extra...))
	return captureStdout(t, func() { require.NoError(t, rootCmd.Execute()) })
}

// An empty `--all` result must keep the list contract: data [] (not null) and
// a limit of at least 1 (#2095, 3P audit 2026-11-13 on CLI 0.10.1).
func TestFetchAllEmptyResultKeepsListContract(t *testing.T) {
	for _, tc := range allListCommands {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":[],"total":0,"limit":` + strconv.Itoa(limit) + `,"offset":0}`))
			}))
			defer srv.Close()

			// --limit 1 is what the auditor passed; --all ignores it.
			out := runAllJSON(t, srv, tc.args, "--limit", "1")

			var raw map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(out), &raw), out)
			assert.JSONEq(t, `[]`, string(raw["data"]), "data must be [], not null")

			var env listEnvelope
			require.NoError(t, json.Unmarshal([]byte(out), &env))
			assert.Equal(t, 0, env.Total)
			assert.Equal(t, tc.pageSize, env.Limit, "empty result reports the page size asked for, not 0")
			assert.Equal(t, 0, env.Offset)
		})
	}
}

// A result spread over several pages comes back as one page holding
// everything: limit is the item count.
func TestFetchAllMultiPageLimitIsItemCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		items := []map[string]any{}
		// 3 items total, served 2 then 1 regardless of the requested limit.
		for i := offset; i < 3 && i < offset+2; i++ {
			items = append(items, map[string]any{"id": "whk_" + strconv.Itoa(i)})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": items, "total": 3, "limit": 2, "offset": offset})
	}))
	defer srv.Close()

	out := runAllJSON(t, srv, []string{"webhooks", "list"})
	var env listEnvelope
	require.NoError(t, json.Unmarshal([]byte(out), &env), out)
	assert.Len(t, env.Data, 3)
	assert.Equal(t, 3, env.Total)
	assert.Equal(t, 3, env.Limit)
	assert.Equal(t, 0, env.Offset)
}

func TestRewrapListNilItems(t *testing.T) {
	body, err := rewrapList(nil, 0, 100)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":[],"total":0,"limit":100,"offset":0}`, string(body))
}
