package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntListValueParsing(t *testing.T) {
	v := &intListValue{}
	require.NoError(t, v.Set(""))
	assert.NotNil(t, v.values)
	assert.Equal(t, []int{}, v.values)

	v = &intListValue{}
	require.NoError(t, v.Set("10, 1440"))
	assert.Equal(t, []int{10, 1440}, v.values)
	assert.Equal(t, "10,1440", v.String())
	assert.Equal(t, "ints", v.Type())

	// A repeated flag appends, as pflag IntSlice did.
	require.NoError(t, v.Set("60"))
	assert.Equal(t, []int{10, 1440, 60}, v.values)

	for _, bad := range []string{"abc", "10,", "10,,20", "1.5"} {
		err := (&intListValue{}).Set(bad)
		require.Error(t, err, bad)
		assert.Contains(t, err.Error(), "invalid reminder offset")
	}
}

func TestRemindersFlagUnsetIsAbsent(t *testing.T) {
	cmd := newEventsCreateCmd()
	v, ok := remindersFlag(cmd, "reminders")
	assert.False(t, ok)
	assert.Nil(t, v)

	_, ok = remindersFlag(cmd, "no-such-flag")
	assert.False(t, ok)

	_, ok = remindersFlag(cmd, "title") // not a reminders flag
	assert.False(t, ok)
}

// rawBodyServer records the raw JSON body of the one request it gets.
func rawBodyServer(t *testing.T, raw *string, reqCount *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		*raw = string(b)
		*reqCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "x_1"})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestReminderFlagsSendEmptyListAndValues(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		field string
	}{
		{"calendars create", []string{"calendars", "create", "--name", "Work"}, "default_reminders"},
		{"calendars update", []string{"calendars", "update", "cal_1"}, "default_reminders"},
		{"events create", []string{"events", "create", "--calendar", "cal_1", "--title", "t", "--start", "2026-12-01T09:00:00Z", "--end", "2026-12-01T10:00:00Z"}, "reminders"},
		{"events update", []string{"events", "update", "evt_1", "--calendar", "cal_1"}, "reminders"},
	}
	flagFor := map[string]string{"default_reminders": "--default-reminders", "reminders": "--reminders"}

	for _, tc := range cases {
		t.Run(tc.name+" empty", func(t *testing.T) {
			var raw string
			var n int
			srv := rawBodyServer(t, &raw, &n)
			args := append(append([]string{}, tc.args...), flagFor[tc.field]+"=")
			require.NoError(t, runCmd(srv.URL, args...))
			assert.Contains(t, raw, `"`+tc.field+`":[]`)
		})

		t.Run(tc.name+" list", func(t *testing.T) {
			var raw string
			var n int
			srv := rawBodyServer(t, &raw, &n)
			args := append(append([]string{}, tc.args...), flagFor[tc.field], "10,1440")
			require.NoError(t, runCmd(srv.URL, args...))
			assert.Contains(t, raw, `"`+tc.field+`":[10,1440]`)
		})

		t.Run(tc.name+" invalid", func(t *testing.T) {
			var raw string
			var n int
			srv := rawBodyServer(t, &raw, &n)
			args := append(append([]string{}, tc.args...), flagFor[tc.field], "abc")
			err := runCmd(srv.URL, args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), `invalid reminder offset "abc"`)
			assert.Zero(t, n, "no request is sent for an invalid value")
		})
	}
}

func TestReminderFlagOmittedLeavesFieldOut(t *testing.T) {
	var raw string
	var n int
	srv := rawBodyServer(t, &raw, &n)
	require.NoError(t, runCmd(srv.URL, "events", "create", "--calendar", "cal_1", "--title", "t", "--start", "2026-12-01T09:00:00Z", "--end", "2026-12-01T10:00:00Z"))
	assert.NotContains(t, raw, "reminders")

	require.NoError(t, runCmd(srv.URL, "calendars", "create", "--name", "Work"))
	assert.NotContains(t, raw, "default_reminders")
}
