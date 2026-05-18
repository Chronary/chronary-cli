package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestICalSub(id, agentID, calID, status string) client.ICalSubscription {
	label := "My Feed"
	return client.ICalSubscription{
		ID:         id,
		AgentID:    agentID,
		CalendarID: calID,
		URL:        "https://example.com/feed.ics",
		Label:      &label,
		Status:     status,
		CreatedAt:  time.Now(),
	}
}

func TestICalListCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/agents/agt_1/ical-subscriptions", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{
			"data":  []client.ICalSubscription{newTestICalSub("ics_1", "agt_1", "cal_1", "active")},
			"total": 1, "limit": 50, "offset": 0,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"ical", "list", "--agent", "agt_1", "--api-key", "chr_sk_test", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestICalListRequiresAgent(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"ical", "list", "--api-key", "chr_sk_test"})
	assert.Error(t, rootCmd.Execute())
}

func TestICalCreateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/v1/agents/agt_1/ical-subscriptions")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "cal_1", body["calendar_id"])
		assert.Equal(t, "https://example.com/feed.ics", body["url"])
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(newTestICalSub("ics_new", "agt_1", "cal_1", "active"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"ical", "create",
		"--agent", "agt_1",
		"--calendar", "cal_1",
		"--url", "https://example.com/feed.ics",
		"--label", "My Feed",
		"--api-key", "chr_sk_test",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestICalGetCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/ical-subscriptions/ics_1", r.URL.Path)
		json.NewEncoder(w).Encode(newTestICalSub("ics_1", "agt_1", "cal_1", "active"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"ical", "get", "ics_1", "--api-key", "chr_sk_test", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestICalUpdateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/v1/ical-subscriptions/ics_1", r.URL.Path)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Label", body["label"])
		json.NewEncoder(w).Encode(newTestICalSub("ics_1", "agt_1", "cal_1", "active"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"ical", "update", "ics_1", "--label", "New Label", "--api-key", "chr_sk_test", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestICalDeleteCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/ical-subscriptions/ics_1", r.URL.Path)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"ical", "delete", "ics_1", "--force", "--api-key", "chr_sk_test", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}

func TestICalSyncCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/ical-subscriptions/ics_1/sync", r.URL.Path)
		w.WriteHeader(202)
		json.NewEncoder(w).Encode(map[string]string{"status": "syncing"})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"ical", "sync", "ics_1", "--api-key", "chr_sk_test", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}
