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

// TestFetchAllPagesPageSizes pins the page size every `--all` command asks
// for to the server's maximum `limit` for that endpoint (#1992). The expected
// values are mirrored by hand from the list query schemas in
// packages/shared/src/schemas: when a cap changes there, change it here and in
// the pageSize* constants. The mock answers 400 above the cap, as the API does.
func TestFetchAllPagesPageSizes(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		path     string
		maxLimit int // server cap; schema named in the comment
	}{
		{"agents", []string{"agents", "list"}, "/v1/agents", 200},                                                        // ListAgentsQuerySchema
		{"calendars", []string{"calendars", "list"}, "/v1/calendars", 200},                                               // ListCalendarsQuerySchema
		{"agent calendars", []string{"calendars", "list", "--agent", "agt_1"}, "/v1/agents/agt_1/calendars", 200},        // ListCalendarsQuerySchema
		{"calendar events", []string{"events", "list", "--calendar", "cal_1"}, "/v1/calendars/cal_1/events", 200},        // ListEventsQuerySchema
		{"agent events", []string{"events", "list", "--agent", "agt_1"}, "/v1/agents/agt_1/events", 200},                 // ListEventsQuerySchema
		{"org events", []string{"events", "list-all"}, "/v1/events", 100},                                                // ListOrgEventsQuerySchema
		{"booking pages", []string{"booking-pages", "list"}, "/v1/booking-pages", 200},                                   // ListBookingPagesQuerySchema
		{"ical subscriptions", []string{"ical", "list", "--agent", "agt_1"}, "/v1/agents/agt_1/ical-subscriptions", 200}, // ListICalSubscriptionsQuerySchema
		{"proposals", []string{"scheduling", "list"}, "/v1/scheduling/proposals", 200},                                   // ListProposalsQuerySchema
		{"webhooks", []string{"webhooks", "list"}, "/v1/webhooks", 100},                                                  // ListWebhooksQuerySchema
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotLimits []int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.path, r.URL.Path)
				limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
				if !assert.NoError(t, err) {
					return
				}
				gotLimits = append(gotLimits, limit)
				if limit > tc.maxLimit {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]any{"type": "validation_error", "message": "Too big: expected number to be <=" + strconv.Itoa(tc.maxLimit)},
					})
					return
				}
				json.NewEncoder(w).Encode(map[string]any{
					"data":  []map[string]any{{"id": "x_1"}},
					"total": 1, "limit": limit, "offset": 0,
				})
			}))
			defer srv.Close()

			rootCmd := NewRootCmd("test")
			args := append(append([]string{}, tc.args...), "--all", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json")
			rootCmd.SetArgs(args)
			require.NoError(t, rootCmd.Execute())
			assert.Equal(t, []int{tc.maxLimit}, gotLimits, "--all must page at the server's max limit")
		})
	}
}
