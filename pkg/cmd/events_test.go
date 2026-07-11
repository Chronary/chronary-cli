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

func newTestEvent(id, title, status string) client.Event {
	return client.Event{
		ID:         id,
		CalendarID: "cal_1",
		Title:      title,
		Status:     status,
		Source:     "internal",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Hour),
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func TestEventsListByCalendar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/calendars/cal_1/events", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []client.Event{newTestEvent("evt_1", "Meeting", "confirmed")},
			"total": 1, "limit": 50, "offset": 0,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "list", "--calendar", "cal_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsListByAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/agents/agt_1/events", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}, "total": 0, "limit": 50, "offset": 0})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "list", "--agent", "agt_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsListRequiresScope(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "list", "--api-key", "chr_sk_xxx"})
	assert.Error(t, rootCmd.Execute())
}

func TestEventsCreateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/calendars/cal_1/events", r.URL.Path)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Standup", body["title"])
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(newTestEvent("evt_new", "Standup", "confirmed"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"events", "create",
		"--calendar", "cal_1",
		"--title", "Standup",
		"--start", "2026-04-12T09:00:00Z",
		"--end", "2026-04-12T09:30:00Z",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsGetCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/calendars/cal_1/events/evt_1", r.URL.Path)
		json.NewEncoder(w).Encode(newTestEvent("evt_1", "Meeting", "confirmed"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "get", "evt_1", "--calendar", "cal_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsUpdateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		json.NewEncoder(w).Encode(newTestEvent("evt_1", "Updated", "confirmed"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "update", "evt_1", "--calendar", "cal_1", "--title", "Updated", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsDeleteCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/calendars/cal_1/events/evt_1", r.URL.Path)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "delete", "evt_1", "--calendar", "cal_1", "--force", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsGetCommandByID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/events/evt_1", r.URL.Path)
		json.NewEncoder(w).Encode(newTestEvent("evt_1", "Meeting", "confirmed"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "get", "evt_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsUpdateCommandByID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/v1/events/evt_1", r.URL.Path)
		json.NewEncoder(w).Encode(newTestEvent("evt_1", "Updated", "confirmed"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "update", "evt_1", "--title", "Updated", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsDeleteCommandByID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/events/evt_1", r.URL.Path)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "delete", "evt_1", "--force", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsCreateHoldCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "hold", body["status"])
		assert.NotEmpty(t, body["hold_expires_at"])
		assert.Equal(t, float64(10), body["hold_priority"])
		w.WriteHeader(201)
		evt := newTestEvent("evt_hold1", "hold slot", "hold")
		json.NewEncoder(w).Encode(evt)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"events", "create",
		"--calendar", "cal_1",
		"--title", "hold slot",
		"--start", "2099-01-01T10:00:00Z",
		"--end", "2099-01-01T10:30:00Z",
		"--status", "hold",
		"--hold-expires-at", "2099-01-01T10:05:00Z",
		"--hold-priority", "10",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsCreateRecurringCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "FREQ=WEEKLY;BYDAY=MO,WE;COUNT=12", body["recurrence_rule"])
		w.WriteHeader(201)
		evt := newTestEvent("evt_rec1", "Standup", "confirmed")
		rule := "FREQ=WEEKLY;BYDAY=MO,WE;COUNT=12"
		evt.RecurrenceRule = &rule
		evt.RecurrenceExdates = []string{}
		json.NewEncoder(w).Encode(evt)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"events", "create",
		"--calendar", "cal_1",
		"--title", "Standup",
		"--start", "2026-04-13T09:00:00Z",
		"--end", "2026-04-13T09:30:00Z",
		"--recurrence-rule", "FREQ=WEEKLY;BYDAY=MO,WE;COUNT=12",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsUpdateRecurrenceRuleCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "FREQ=DAILY;COUNT=5", body["recurrence_rule"])
		json.NewEncoder(w).Encode(newTestEvent("evt_rec1", "Standup", "confirmed"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "update", "evt_rec1", "--recurrence-rule", "FREQ=DAILY;COUNT=5", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsUpdateClearRecurrenceRuleCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		// --recurrence-rule "" must send an explicit JSON null (clear the rule).
		v, present := body["recurrence_rule"]
		assert.True(t, present, "expected recurrence_rule key in payload")
		assert.Nil(t, v)
		json.NewEncoder(w).Encode(newTestEvent("evt_rec1", "Standup", "confirmed"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "update", "evt_rec1", "--recurrence-rule", "", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsListExpandCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/calendars/cal_1/events", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, "true", q.Get("expand"))
		assert.Equal(t, "2026-04-13T00:00:00Z", q.Get("start_after"))
		assert.Equal(t, "2026-05-13T00:00:00Z", q.Get("start_before"))
		evt := newTestEvent("evt_rec1", "Standup", "confirmed")
		evt.RecurringEventID = "evt_rec1"
		evt.OriginalStartTime = "2026-04-15T09:00:00Z"
		json.NewEncoder(w).Encode(map[string]any{
			"data": []client.Event{evt}, "total": 1, "limit": 50, "offset": 0,
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"events", "list", "--calendar", "cal_1",
		"--expand",
		"--start-after", "2026-04-13T00:00:00Z",
		"--start-before", "2026-05-13T00:00:00Z",
		"--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsListExpandRequiresWindow(t *testing.T) {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "list", "--calendar", "cal_1", "--expand", "--api-key", "chr_sk_xxx"})
	assert.Error(t, rootCmd.Execute())
}

func TestEventsDeleteOccurrenceCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/calendars/cal_1/events/evt_rec1", r.URL.Path)
		assert.Equal(t, "2026-04-15T09:00:00Z", r.URL.Query().Get("occurrence_start"))
		// Occurrence-cancel returns 200 with the updated series master, not 204.
		evt := newTestEvent("evt_rec1", "Standup", "confirmed")
		rule := "FREQ=WEEKLY;BYDAY=MO,WE;COUNT=12"
		evt.RecurrenceRule = &rule
		evt.RecurrenceExdates = []string{"2026-04-15T09:00:00Z"}
		json.NewEncoder(w).Encode(evt)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"events", "delete", "evt_rec1", "--calendar", "cal_1",
		"--occurrence-start", "2026-04-15T09:00:00Z",
		"--force", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsDeleteOccurrenceCommandByID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/events/evt_rec1", r.URL.Path)
		assert.Equal(t, "2026-04-15T09:00:00Z", r.URL.Query().Get("occurrence_start"))
		evt := newTestEvent("evt_rec1", "Standup", "confirmed")
		evt.RecurrenceExdates = []string{"2026-04-15T09:00:00Z"}
		json.NewEncoder(w).Encode(evt)
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"events", "delete", "evt_rec1",
		"--occurrence-start", "2026-04-15T09:00:00Z",
		"--force", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsConfirmCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v1/events/evt_hold1/confirm", r.URL.Path)
		json.NewEncoder(w).Encode(newTestEvent("evt_hold1", "confirmed hold", "confirmed"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "confirm", "evt_hold1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestEventsReleaseCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v1/events/evt_hold1/release", r.URL.Path)
		json.NewEncoder(w).Encode(newTestEvent("evt_hold1", "released hold", "cancelled"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"events", "release", "evt_hold1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}
