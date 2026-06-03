package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvailabilityAgentCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v1/agents/agt_1/availability")
		assert.NotEmpty(t, r.URL.Query().Get("start"))
		assert.NotEmpty(t, r.URL.Query().Get("end"))
		json.NewEncoder(w).Encode(map[string]any{
			"slots": []map[string]string{
				{"start": "2026-04-12T09:00:00Z", "end": "2026-04-12T09:30:00Z"},
				{"start": "2026-04-12T10:00:00Z", "end": "2026-04-12T10:30:00Z"},
			},
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"availability", "agent", "agt_1",
		"--start", "2026-04-12T00:00:00Z",
		"--end", "2026-04-12T23:59:59Z",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestAvailabilityCalendarCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v1/calendars/cal_1/availability")
		json.NewEncoder(w).Encode(map[string]any{"slots": []any{}})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"availability", "calendar", "cal_1",
		"--start", "2026-04-12T00:00:00Z",
		"--end", "2026-04-12T23:59:59Z",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestAvailabilityCrossCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/availability", r.URL.Path)
		assert.Equal(t, "agt_1,agt_2", r.URL.Query().Get("agents"))
		json.NewEncoder(w).Encode(map[string]any{
			"slots": []map[string]string{
				{"start": "2026-04-12T14:00:00Z", "end": "2026-04-12T14:30:00Z"},
			},
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"availability", "cross",
		"--agents", "agt_1,agt_2",
		"--start", "2026-04-12T00:00:00Z",
		"--end", "2026-04-12T23:59:59Z",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}

func TestAvailabilityWithSlotDuration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "1h", r.URL.Query().Get("slot_duration"))
		assert.Equal(t, "true", r.URL.Query().Get("include_busy"))
		json.NewEncoder(w).Encode(map[string]any{
			"slots": []any{},
			"busy":  []map[string]string{{"start": "2026-04-12T09:00:00Z", "end": "2026-04-12T10:00:00Z"}},
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{
		"availability", "agent", "agt_1",
		"--start", "2026-04-12T00:00:00Z",
		"--end", "2026-04-12T23:59:59Z",
		"--slot-duration", "1h",
		"--include-busy",
		"--api-key", "chr_sk_xxx",
		"--base-url", srv.URL,
		"--output", "json",
	})
	require.NoError(t, rootCmd.Execute())
}
