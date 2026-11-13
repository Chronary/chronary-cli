package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordedRequest is one request the mock API received.
type recordedRequest struct {
	Method string
	Path   string
	Body   map[string]any
}

// newRecordingServer answers every request with 201 {"id": "x_1"} and records
// what it received.
func newRecordingServer(t *testing.T) (*httptest.Server, *[]recordedRequest) {
	t.Helper()
	var reqs []recordedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		var body map[string]any
		if len(raw) > 0 {
			assert.NoError(t, json.Unmarshal(raw, &body))
		}
		reqs = append(reqs, recordedRequest{Method: r.Method, Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": "x_1"})
	}))
	t.Cleanup(srv.Close)
	return srv, &reqs
}

func runCmd(srvURL string, args ...string) error {
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs(append(args, "--api-key", "chr_sk_xxx", "--base-url", srvURL, "--output", "json"))
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	return rootCmd.Execute()
}

// createCases are the `create [@file]` commands whose body fields used to be
// cobra-required flags, so @file / @- failed before RunE ran (#1993).
var createCases = []struct {
	name     string
	args     []string // command plus non-body flags (path params)
	path     string
	fileBody map[string]any
	// flagsMissing passes every required body flag but one; wantMissing names it.
	flagsMissing []string
	wantMissing  string
}{
	{
		name:         "agents",
		args:         []string{"agents", "create"},
		path:         "/v1/agents",
		fileBody:     map[string]any{"name": "Bot", "type": "ai"},
		flagsMissing: []string{"--type", "ai"},
		wantMissing:  `required flag(s) "name" not set`,
	},
	{
		name:         "calendars",
		args:         []string{"calendars", "create", "--agent", "agt_1"},
		path:         "/v1/agents/agt_1/calendars",
		fileBody:     map[string]any{"name": "Work", "timezone": "UTC"},
		flagsMissing: []string{"--timezone", "UTC"},
		wantMissing:  `required flag(s) "name" not set`,
	},
	{
		name:         "booking-pages",
		args:         []string{"booking-pages", "create"},
		path:         "/v1/booking-pages",
		fileBody:     map[string]any{"calendar_id": "cal_1", "title": "Intro call"},
		flagsMissing: []string{"--calendar", "cal_1"},
		wantMissing:  `required flag(s) "title" not set`,
	},
	{
		name:         "events",
		args:         []string{"events", "create", "--calendar", "cal_1"},
		path:         "/v1/calendars/cal_1/events",
		fileBody:     map[string]any{"title": "Standup", "start_time": "2026-12-01T09:00:00Z", "end_time": "2026-12-01T09:15:00Z"},
		flagsMissing: []string{"--title", "Standup"},
		wantMissing:  `required flag(s) "end", "start" not set`,
	},
	{
		name:         "webhooks",
		args:         []string{"webhooks", "create"},
		path:         "/v1/webhooks",
		fileBody:     map[string]any{"url": "https://example.com/hook", "events": []any{"event.created"}},
		flagsMissing: []string{"--url", "https://example.com/hook"},
		wantMissing:  `required flag(s) "events" not set`,
	},
}

func TestCreateFromFileWithoutBodyFlags(t *testing.T) {
	for _, tc := range createCases {
		t.Run(tc.name, func(t *testing.T) {
			srv, reqs := newRecordingServer(t)
			fileArg := writeTempJSON(t, tc.name+".json", tc.fileBody)

			args := append(append([]string{}, tc.args...), fileArg)
			require.NoError(t, runCmd(srv.URL, args...))

			require.Len(t, *reqs, 1)
			assert.Equal(t, "POST", (*reqs)[0].Method)
			assert.Equal(t, tc.path, (*reqs)[0].Path)
			assert.Equal(t, tc.fileBody, (*reqs)[0].Body, "the @file body is sent unchanged")
		})
	}
}

func TestCreateFlagsPathStillRequiresBodyFlags(t *testing.T) {
	for _, tc := range createCases {
		t.Run(tc.name, func(t *testing.T) {
			srv, reqs := newRecordingServer(t)

			args := append(append([]string{}, tc.args...), tc.flagsMissing...)
			err := runCmd(srv.URL, args...)
			require.Error(t, err)
			assert.Equal(t, tc.wantMissing, err.Error())
			assert.Empty(t, *reqs, "no request is sent when a required flag is missing")
		})
	}
}

func TestCreateFromUnreadableFileReturnsReadError(t *testing.T) {
	for _, tc := range createCases {
		t.Run(tc.name, func(t *testing.T) {
			srv, reqs := newRecordingServer(t)

			args := append(append([]string{}, tc.args...), "@does-not-exist.json")
			err := runCmd(srv.URL, args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), `reading file "does-not-exist.json"`)
			assert.Empty(t, *reqs)
		})
	}
}

func TestCreateFromInvalidJSONFileReturnsParseError(t *testing.T) {
	srv, reqs := newRecordingServer(t)
	path := t.TempDir() + "/bad.json"
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

	err := runCmd(srv.URL, "agents", "create", "@"+path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing JSON")
	assert.Empty(t, *reqs)
}

// The exact repro from #1993: body on stdin via @-.
func TestCalendarsCreateFromStdin(t *testing.T) {
	srv, reqs := newRecordingServer(t)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString(`{"name":"x","timezone":"UTC"}`)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin; r.Close() })

	require.NoError(t, runCmd(srv.URL, "calendars", "create", "@-", "--agent", "agt_1"))

	require.Len(t, *reqs, 1)
	assert.Equal(t, "/v1/agents/agt_1/calendars", (*reqs)[0].Path)
	assert.Equal(t, map[string]any{"name": "x", "timezone": "UTC"}, (*reqs)[0].Body)
}

func TestEventsCreateStillRequiresCalendarWithFile(t *testing.T) {
	srv, reqs := newRecordingServer(t)
	fileArg := writeTempJSON(t, "event.json", map[string]any{"title": "x"})

	err := runCmd(srv.URL, "events", "create", fileArg)
	require.Error(t, err)
	assert.Equal(t, `required flag(s) "calendar" not set`, err.Error())
	assert.Empty(t, *reqs)
}

func TestRequireFlagsSet(t *testing.T) {
	cmd := newEventsCreateCmd()
	require.NoError(t, cmd.Flags().Set("title", "x"))
	assert.EqualError(t, requireFlagsSet(cmd, "title", "start", "end"), `required flag(s) "end", "start" not set`)

	require.NoError(t, cmd.Flags().Set("start", "a"))
	require.NoError(t, cmd.Flags().Set("end", "b"))
	assert.NoError(t, requireFlagsSet(cmd, "title", "start", "end"))

	// An explicitly empty value counts as set, as with cobra's MarkFlagRequired.
	require.NoError(t, cmd.Flags().Set("description", ""))
	assert.NoError(t, requireFlagsSet(cmd, "description"))
}
