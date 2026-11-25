package cmd

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every delete command that confirms before sending its DELETE (#2692).
var deleteCommands = []struct {
	name string
	args []string
}{
	{"keys", []string{"keys", "delete", "key_1"}},
	{"agents", []string{"agents", "delete", "agt_1"}},
	{"availability-rules", []string{"calendars", "availability-rules", "delete", "cal_1"}},
	{"calendars", []string{"calendars", "delete", "cal_1"}},
	{"events", []string{"events", "delete", "evt_1", "--calendar", "cal_1"}},
	{"webhooks", []string{"webhooks", "delete", "wh_1"}},
	{"ical", []string{"ical", "delete", "ics_1"}},
	{"booking-pages", []string{"booking-pages", "delete", "bkp_1"}},
}

func stubConfirmPrompt(t *testing.T, confirm bool, err error) {
	t.Helper()
	orig := runConfirmPrompt
	runConfirmPrompt = func(string, string) (bool, error) { return confirm, err }
	t.Cleanup(func() { runConfirmPrompt = orig })
	stubStdinTerminal(t, true)
}

func stubStdinTerminal(t *testing.T, isTTY bool) {
	t.Helper()
	orig := stdinIsTerminal
	stdinIsTerminal = func() bool { return isTTY }
	t.Cleanup(func() { stdinIsTerminal = orig })
}

// runDelete runs one delete command against a server that counts DELETEs.
func runDelete(t *testing.T, args []string, extra ...string) (string, error, int32) {
	t.Helper()
	var deletes int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			atomic.AddInt32(&deletes, 1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	var execErr error
	out := captureStdout(t, func() {
		rootCmd := NewRootCmd("test")
		full := append(append([]string{}, args...), "--api-key", "chr_sk_xxx", "--base-url", srv.URL)
		rootCmd.SetArgs(append(full, extra...))
		execErr = rootCmd.Execute()
	})
	return out, execErr, atomic.LoadInt32(&deletes)
}

func TestDeleteWithoutTerminalFailsAndSendsNothing(t *testing.T) {
	for _, tc := range deleteCommands {
		t.Run(tc.name, func(t *testing.T) {
			stubConfirmPrompt(t, false, errors.New("could not open a new TTY: open /dev/tty: no such device"))
			out, err, deletes := runDelete(t, tc.args)
			require.Error(t, err, "a prompt failure must exit non-zero")
			assert.Contains(t, err.Error(), "--yes")
			assert.Contains(t, err.Error(), "no request was sent")
			assert.NotContains(t, out, "Cancelled.")
			assert.Equal(t, int32(0), deletes)
		})
	}
}

func TestDeleteAbortOrNoPrintsCancelledAndExitsZero(t *testing.T) {
	cases := []struct {
		label   string
		confirm bool
		err     error
	}{
		{"ctrl-c", false, huh.ErrUserAborted},
		{"answered no", false, nil},
	}
	for _, tc := range deleteCommands {
		for _, c := range cases {
			t.Run(tc.name+"/"+c.label, func(t *testing.T) {
				stubConfirmPrompt(t, c.confirm, c.err)
				out, err, deletes := runDelete(t, tc.args)
				require.NoError(t, err)
				assert.Contains(t, out, "Cancelled.")
				assert.Equal(t, int32(0), deletes)
			})
		}
	}
}

func TestDeleteConfirmedOrYesSendsOneDelete(t *testing.T) {
	for _, tc := range deleteCommands {
		t.Run(tc.name+"/confirmed", func(t *testing.T) {
			stubConfirmPrompt(t, true, nil)
			_, err, deletes := runDelete(t, tc.args)
			require.NoError(t, err)
			assert.Equal(t, int32(1), deletes)
		})
		t.Run(tc.name+"/--yes", func(t *testing.T) {
			stubConfirmPrompt(t, false, errors.New("prompt must not run with --yes"))
			_, err, deletes := runDelete(t, tc.args, "--yes")
			require.NoError(t, err)
			assert.Equal(t, int32(1), deletes)
		})
	}
}

// TERM=dumb makes huh read a non-terminal stdin's EOF as "no" with no error,
// so stdin is checked before prompting (#2692 review).
func TestDeleteWithNonTerminalStdinFailsWithoutPrompting(t *testing.T) {
	for _, tc := range deleteCommands {
		t.Run(tc.name, func(t *testing.T) {
			prompted := false
			orig := runConfirmPrompt
			runConfirmPrompt = func(string, string) (bool, error) { prompted = true; return false, nil }
			t.Cleanup(func() { runConfirmPrompt = orig })
			stubStdinTerminal(t, false)

			out, err, deletes := runDelete(t, tc.args)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "--yes")
			assert.Contains(t, err.Error(), "no request was sent")
			assert.NotContains(t, out, "Cancelled.")
			assert.False(t, prompted, "no prompt when stdin is not a terminal")
			assert.Equal(t, int32(0), deletes)
		})
	}
}
