package cmd

import (
	"fmt"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
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

// requireAPIKey returns an error if the client has no API key configured.
func requireAPIKey(c *client.Client) error {
	if c.APIKey == "" {
		return fmt.Errorf("no API key configured. Run `chronary auth login` or set CHRONARY_API_KEY")
	}
	return nil
}

// printStructured prints data as JSON or YAML if the format is not table.
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
