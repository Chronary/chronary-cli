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

func newTestCalendar(id, name, tz string) client.Calendar {
	return client.Calendar{
		ID:        id,
		Name:      name,
		Timezone:  tz,
		Metadata:  json.RawMessage(`{}`),
		ICalURL:   "https://api.chronary.ai/ical/token123.ics",
		CreatedAt: time.Now(),
	}
}

func TestCalendarsListCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/calendars", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []client.Calendar{newTestCalendar("cal_1", "Work", "America/New_York")},
			"total": 1, "limit": 50, "offset": 0,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"calendars", "list", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestCalendarsListByAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/agents/agt_1/calendars", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}, "total": 0, "limit": 50, "offset": 0})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"calendars", "list", "--agent", "agt_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestCalendarsCreateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Work", body["name"])
		assert.Equal(t, "America/New_York", body["timezone"])
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(newTestCalendar("cal_new", "Work", "America/New_York"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"calendars", "create", "--name", "Work", "--timezone", "America/New_York", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestCalendarsGetCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/calendars/cal_1", r.URL.Path)
		json.NewEncoder(w).Encode(newTestCalendar("cal_1", "Work", "UTC"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"calendars", "get", "cal_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestCalendarsUpdateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		json.NewEncoder(w).Encode(newTestCalendar("cal_1", "Renamed", "UTC"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"calendars", "update", "cal_1", "--name", "Renamed", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestCalendarsDeleteCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"calendars", "delete", "cal_1", "--force", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}

func TestCalendarsContextCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/calendars/cal_1/context", r.URL.Path)
		_ = json.NewEncoder(w).Encode(client.CalendarContext{
			CalendarID:   "cal_1",
			Now:          "2026-04-16T12:00:00Z",
			AgentStatus:  "idle",
			RecentEvents: []client.Event{},
			Upcoming:     []client.Event{},
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"calendars", "context", "cal_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestCalendarsAvailabilityRulesGetCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/calendars/cal_1/availability-rules", r.URL.Path)
		_ = json.NewEncoder(w).Encode(client.AvailabilityRules{
			ID:                  "rul_1",
			CalendarID:          "cal_1",
			BufferBeforeMinutes: 0,
			BufferAfterMinutes:  0,
			Timezone:            "UTC",
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"calendars", "availability-rules", "get", "cal_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestCalendarsAvailabilityRulesSetCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v1/calendars/cal_1/availability-rules", r.URL.Path)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, float64(10), body["buffer_before_minutes"])
		_ = json.NewEncoder(w).Encode(client.AvailabilityRules{
			ID: "rul_1", CalendarID: "cal_1",
			BufferBeforeMinutes: 10, BufferAfterMinutes: 5,
			Timezone: "UTC", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
	}))
	defer srv.Close()

	fileArg := writeTempJSON(t, "rules.json", map[string]any{
		"buffer_before_minutes": 10,
		"buffer_after_minutes":  5,
		"timezone":              "UTC",
	})

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"calendars", "availability-rules", "set", "cal_1", fileArg,
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestCalendarsAvailabilityRulesDeleteCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/calendars/cal_1/availability-rules", r.URL.Path)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"calendars", "availability-rules", "delete", "cal_1", "--force", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}
