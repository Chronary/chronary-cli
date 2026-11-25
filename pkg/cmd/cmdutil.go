package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/config"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// formatError returns a user-friendly error for display.
func formatError(err error) error {
	switch e := err.(type) {
	case *client.APIError:
		return fmt.Errorf("%s", e.FriendlyMessage())
	case *client.HTTPError:
		return fmt.Errorf("%s", e.FriendlyMessage())
	default:
		return err
	}
}

// runConfirmPrompt shows the yes/no prompt for a destructive command. A
// package variable so tests can stub it: a real prompt hangs a test run that
// has a terminal and fails one that has none.
var runConfirmPrompt = func(title, description string) (bool, error) {
	var confirm bool
	err := huh.NewConfirm().
		Title(title).
		Description(description).
		Value(&confirm).
		Run()
	return confirm, err
}

// stdinIsTerminal reports whether stdin is an interactive terminal. A package
// variable so tests can stub it.
var stdinIsTerminal = func() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// confirmDestructive asks before a delete. It returns proceed=true when
// --force/--yes is set or the user confirms. A "no" or Ctrl-C prints
// "Cancelled." and returns proceed=false with no error (exit 0). Any other
// prompt failure, typically no terminal in CI, cron or an agent sandbox, is an
// error (exit 1): it used to print "Cancelled." and exit 0, so automation
// believed a delete had happened when no request was sent (#2692).
func confirmDestructive(cmd *cobra.Command, title, description string) (bool, error) {
	force, _ := cmd.Flags().GetBool("force")
	yes, _ := cmd.Flags().GetBool("yes")
	if force || yes {
		return true, nil
	}
	// huh falls back to an accessible line prompt under TERM=dumb, which
	// reads EOF from a non-terminal stdin as the default "no" and returns no
	// error, so a missing terminal looked like a user's "no". Refuse up front
	// when stdin is not a terminal.
	if !stdinIsTerminal() {
		return false, fmt.Errorf("confirmation required but stdin is not a terminal. Re-run with --yes to confirm; no request was sent")
	}
	confirm, err := runConfirmPrompt(title, description)
	if errors.Is(err, huh.ErrUserAborted) || (err == nil && !confirm) {
		fmt.Println("Cancelled.")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("confirmation required but no prompt could be shown (%v). Re-run with --yes to confirm; no request was sent", err)
	}
	return true, nil
}

// requireAPIKey returns an error if the client has no API key configured.
func requireAPIKey(c *client.Client) error {
	if c.APIKey == "" {
		// A corrupt config used to read as "no API key configured", which sent
		// users to `auth login`, which then overwrote every profile (#2562).
		// Name the file instead. Only commands that need a key get here, so
		// key-less commands (health, plans, signup) keep working.
		if path, err := config.Path(); err == nil {
			if _, loadErr := config.LoadFrom(path); loadErr != nil {
				return fmt.Errorf("config file %s could not be read: %w. Fix or move it, or pass --api-key / CHRONARY_API_KEY", path, loadErr)
			}
		}
		return fmt.Errorf("no API key configured. Run `chronary auth login` or set CHRONARY_API_KEY")
	}
	return nil
}

// printBody prints an API response body as JSON or YAML if the format is not
// table. It prints the body the API sent, not a re-marshalled client struct:
// the structs hold only what the table needs and drift from the API, so
// printing them dropped fields and reformatted values (#2738). Returns true if
// it printed (caller should return), false if table rendering is needed.
func printBody(cmd *cobra.Command, body []byte) bool {
	switch outputFormat(cmd) {
	case "json":
		output.PrintRawJSON(body)
		return true
	case "yaml":
		output.PrintRawYAML(body)
		return true
	}
	return false
}

// printStructured prints locally built data (no API response body, e.g. auth
// status or version) as JSON or YAML if the format is not table. Commands that
// print an API response use printBody.
// Returns true if it printed (caller should return), false if table rendering needed.
func printStructured(cmd *cobra.Command, data any) bool {
	format := outputFormat(cmd)
	switch format {
	case "json":
		output.PrintJSON(data)
		return true
	case "yaml":
		output.PrintYAML(data)
		return true
	}
	return false
}
