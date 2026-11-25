package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bughunt-BH-080 (#2691): with NOC_QUERY_CAPS on, every list rejects
// offset > 10000 with 400 validation_error. `--all` paged by offset only, so
// it failed partway past 10,000 rows, even on lists that return next_cursor.
// It now follows next_cursor where the list has one.

type cappedListAPI struct {
	rows       int
	withCursor bool
	mu         sync.Mutex
	offsets    []int
	cursors    []string
}

func (a *cappedListAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	cursor, hasCursor := q.Get("cursor"), q.Has("cursor")
	w.Header().Set("Content-Type", "application/json")
	if offset > 10000 || (hasCursor && (!a.withCursor || offset > 0)) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"type":"validation_error","message":"offset must be at most 10000","param":"offset"}}`))
		return
	}
	a.mu.Lock()
	a.offsets = append(a.offsets, offset)
	if hasCursor {
		a.cursors = append(a.cursors, cursor)
	}
	a.mu.Unlock()

	start := offset
	if hasCursor {
		raw, _ := base64.StdEncoding.DecodeString(cursor)
		start, _ = strconv.Atoi(string(raw))
	}
	stop := min(start+limit, a.rows)
	data := make([]map[string]any, 0, stop-start)
	for i := start; i < stop; i++ {
		data = append(data, map[string]any{"id": fmt.Sprintf("row_%d", i)})
	}
	body := map[string]any{"data": data, "total": a.rows, "limit": limit, "offset": offset}
	if a.withCursor {
		if stop < a.rows {
			body["next_cursor"] = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(stop)))
		} else {
			body["next_cursor"] = nil
		}
	}
	_ = json.NewEncoder(w).Encode(body)
}

// runAllJSONToFile is runAllJSON with stdout sent to a file: 12,000 rows are
// more than a pipe buffers (as are a few hundred events), and captureStdout reads only after the command ends.
func runAllJSONToFile(t *testing.T, srv *httptest.Server, args []string) string {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "stdout.json"))
	require.NoError(t, err)
	orig := os.Stdout
	os.Stdout = f
	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs(append(append([]string{}, args...), "--all", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"))
	execErr := rootCmd.Execute()
	os.Stdout = orig
	require.NoError(t, f.Close())
	require.NoError(t, execErr)
	b, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	return string(b)
}

func TestFetchAllFollowsNextCursorPastOffsetCap(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"org events", []string{"events", "list-all"}},
		{"agents", []string{"agents", "list"}},
		{"calendars", []string{"calendars", "list"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &cappedListAPI{rows: 12000, withCursor: true}
			srv := httptest.NewServer(api)
			defer srv.Close()

			out := runAllJSONToFile(t, srv, tc.args)

			var got listEnvelope
			require.NoError(t, json.Unmarshal([]byte(out), &got))
			assert.Len(t, got.Data, 12000)
			assert.Equal(t, 12000, got.Total)
			for _, off := range api.offsets {
				assert.Equal(t, 0, off, "cursor paging sends no offset")
			}
			assert.NotEmpty(t, api.cursors)
		})
	}
}

func TestFetchAllOffsetOnlyListNeverSendsCursor(t *testing.T) {
	api := &cappedListAPI{rows: 450, withCursor: false}
	srv := httptest.NewServer(api)
	defer srv.Close()

	out := runAllJSONToFile(t, srv, []string{"events", "list", "--calendar", "cal_1"})

	var got listEnvelope
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Len(t, got.Data, 450)
	assert.Empty(t, api.cursors)
	assert.Equal(t, []int{0, 200, 400}, api.offsets)
}

// Review of #2748: a list that hands back the cursor just sent must not loop.
func TestFetchAllStopsOnRepeatedCursor(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"a"}],"total":5,"limit":1,"offset":0,"next_cursor":"same"}`))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"agents", "list", "--all", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same next_cursor")
	assert.Equal(t, 2, calls)
}
