package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #2519: an ID given on the command line (or in an ID flag) stays inside one
// escaped path segment, and "", "." and ".." are refused before any request.

const cliEscapeID = "../keys#x?y"
const cliEscapeEncoded = "..%2Fkeys%23x%3Fy"

func TestCommandsKeepIDsInOneSegment(t *testing.T) {
	type tc struct {
		args   []string // "ID" is replaced with the value under test
		prefix string
	}
	cases := []tc{
		{[]string{"agents", "get", "ID"}, "/v1/agents/"},
		{[]string{"agents", "delete", "ID", "--yes"}, "/v1/agents/"},
		{[]string{"billing", "upgrade-request", "get", "ID"}, "/v1/billing/upgrade-requests/"},
		{[]string{"billing", "upgrade-request", "cancel", "ID"}, "/v1/billing/upgrade-requests/"},
		{[]string{"booking-pages", "get", "ID"}, "/v1/booking-pages/"},
		{[]string{"booking-pages", "rotate-slug", "ID"}, "/v1/booking-pages/"},
		{[]string{"booking-pages", "bookings", "ID"}, "/v1/booking-pages/"},
		{[]string{"calendars", "get", "ID"}, "/v1/calendars/"},
		{[]string{"calendars", "context", "ID"}, "/v1/calendars/"},
		{[]string{"calendars", "rotate-ical-token", "ID"}, "/v1/calendars/"},
		{[]string{"calendars", "availability-rules", "get", "ID"}, "/v1/calendars/"},
		{[]string{"calendars", "list", "--agent", "ID"}, "/v1/agents/"},
		{[]string{"connection-links", "get", "ID"}, "/v1/connection-links/"},
		{[]string{"connection-links", "cancel", "ID"}, "/v1/connection-links/"},
		{[]string{"events", "get", "ID"}, "/v1/events/"},
		{[]string{"events", "get", "evt_1", "--calendar", "ID"}, "/v1/calendars/"},
		{[]string{"events", "confirm", "ID"}, "/v1/events/"},
		{[]string{"events", "release", "ID"}, "/v1/events/"},
		{[]string{"events", "list", "--calendar", "ID"}, "/v1/calendars/"},
		{[]string{"events", "list", "--agent", "ID"}, "/v1/agents/"},
		{[]string{"ical", "get", "ID"}, "/v1/ical-subscriptions/"},
		{[]string{"ical", "sync", "ID"}, "/v1/ical-subscriptions/"},
		{[]string{"ical", "pause", "ID"}, "/v1/ical-subscriptions/"},
		{[]string{"ical", "resume", "ID"}, "/v1/ical-subscriptions/"},
		{[]string{"scheduling", "get", "ID"}, "/v1/scheduling/proposals/"},
		{[]string{"scheduling", "resolve", "ID"}, "/v1/scheduling/proposals/"},
		{[]string{"webhooks", "get", "ID"}, "/v1/webhooks/"},
		{[]string{"webhooks", "deliveries", "ID"}, "/v1/webhooks/"},
		{[]string{"webhooks", "delivery", "ID", "whd_1"}, "/v1/webhooks/"},
		{[]string{"webhooks", "delivery", "whk_1", "ID"}, "/v1/webhooks/whk_1/deliveries/"},
		{[]string{"webhooks", "replay", "whk_1", "ID"}, "/v1/webhooks/whk_1/deliveries/"},
	}

	run := func(t *testing.T, args []string, id string) ([]string, error) {
		var mu sync.Mutex
		var got []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			got = append(got, r.URL.EscapedPath())
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[],"total":0,"limit":50,"offset":0}`))
		}))
		defer srv.Close()
		full := make([]string, 0, len(args)+6)
		for _, a := range args {
			if a == "ID" {
				a = id
			}
			full = append(full, a)
		}
		full = append(full, "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json")
		rootCmd := NewRootCmd("test")
		rootCmd.SetArgs(full)
		err := rootCmd.Execute()
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...), err
	}

	for _, c := range cases {
		name := strings.Join(c.args, " ")
		t.Run(name, func(t *testing.T) {
			got, _ := run(t, c.args, cliEscapeID) // the stub's body may not parse; only the path matters
			require.NotEmpty(t, got)
			assert.True(t, strings.HasPrefix(got[0], c.prefix+cliEscapeEncoded), "path %q, want prefix %q", got[0], c.prefix+cliEscapeEncoded)

			got, err := run(t, c.args, "..")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "No request was sent")
			assert.Empty(t, got)
		})
	}
}
