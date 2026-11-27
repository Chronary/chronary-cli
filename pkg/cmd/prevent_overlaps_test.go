package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #595: --prevent-overlaps on calendars create and update is sent only when
// the flag is given, as a JSON boolean.
func TestPreventOverlapsFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want any // nil = field absent
	}{
		{"create on", []string{"calendars", "create", "--name", "Work", "--prevent-overlaps"}, true},
		{"create explicit off", []string{"calendars", "create", "--name", "Work", "--prevent-overlaps=false"}, false},
		{"create omitted", []string{"calendars", "create", "--name", "Work"}, nil},
		{"update on", []string{"calendars", "update", "cal_1", "--prevent-overlaps"}, true},
		{"update off", []string{"calendars", "update", "cal_1", "--prevent-overlaps=false"}, false},
		{"update omitted", []string{"calendars", "update", "cal_1", "--name", "n"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var raw string
			var n int
			srv := rawBodyServer(t, &raw, &n)
			require.NoError(t, runCmd(srv.URL, tc.args...))
			require.Equal(t, 1, n)
			var body map[string]any
			require.NoError(t, json.Unmarshal([]byte(raw), &body))
			v, ok := body["prevent_overlaps"]
			if tc.want == nil {
				assert.False(t, ok, "prevent_overlaps must be absent, body %s", raw)
				return
			}
			assert.Equal(t, tc.want, v)
		})
	}
}

func TestPreventOverlapsAloneSatisfiesUpdate(t *testing.T) {
	var raw string
	var n int
	srv := rawBodyServer(t, &raw, &n)
	require.NoError(t, runCmd(srv.URL, "calendars", "update", "cal_1", "--prevent-overlaps=false"))
	assert.JSONEq(t, `{"prevent_overlaps":false}`, raw)

	err := runCmd(srv.URL, "calendars", "update", "cal_1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--prevent-overlaps")
}
