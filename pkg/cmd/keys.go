package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

func newKeysCmd() *cobra.Command {
	keysCmd := &cobra.Command{
		Use:     "keys",
		Aliases: []string{"key"},
		Short:   "Manage agent-scoped API keys",
	}

	keysCmd.AddCommand(newKeysListCmd())
	keysCmd.AddCommand(newKeysCreateCmd())
	keysCmd.AddCommand(newKeysDeleteCmd())

	return keysCmd
}

func newKeysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List agent-scoped API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Get("/v1/keys")
			if err != nil {
				return formatError(err)
			}

			var resp client.ScopedAPIKeyListResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)
			rows := make([][]string, len(resp.Keys))
			for i, key := range resp.Keys {
				label := "-"
				if key.Label != nil && *key.Label != "" {
					label = *key.Label
				}
				rows[i] = []string{
					key.ID,
					key.AgentID,
					label,
					key.KeyPrefix,
					key.CreatedAt.Format("2006-01-02 15:04"),
					formatKeyExpiry(key.ExpiresAt, time.Now()),
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "Agent", "Label", "Prefix", "Created", "Expires"},
				Rows:    rows,
			}, nc)

			return nil
		},
	}
}

func newKeysCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [@file]",
		Short: "Create an agent-scoped API key",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			payload, err := checkFileArg(args, 0)
			if err != nil {
				return err
			}

			if payload == nil {
				agentID, _ := cmd.Flags().GetString("agent")
				if agentID == "" {
					return fmt.Errorf("--agent is required unless using @file")
				}

				payload = map[string]any{
					"agent_id": agentID,
				}

				if cmd.Flags().Changed("label") {
					label, _ := cmd.Flags().GetString("label")
					if label == "" {
						return fmt.Errorf("--label cannot be empty")
					}
					payload["label"] = label
				}

				if cmd.Flags().Changed("expires-in") {
					expiresIn, _ := cmd.Flags().GetString("expires-in")
					if !validKeyExpiry(expiresIn) {
						return fmt.Errorf("--expires-in must be one of %s", strings.Join(keyExpiryOptions, ", "))
					}
					payload["expires_in"] = expiresIn
				}
			}

			body, _, err := c.Post("/v1/keys", payload)
			if err != nil {
				return formatError(err)
			}

			var created client.CreatedScopedAPIKey
			if err := json.Unmarshal(body, &created); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			// Structured output carries the new key on purpose: agents parse
			// it from stdout, and the API never returns it again.
			if printStructured(cmd, created) {
				return nil
			}

			label := "-"
			if created.Label != nil && *created.Label != "" {
				label = *created.Label
			}

			fmt.Printf("Created agent-scoped key %s for %s.\n", created.ID, created.AgentID)
			// Intentional one-time display of a newly created key: the API
			// returns the full key only in this response, so this stdout line
			// is the caller's only copy. It is printed once, only on stdout, and
			// never in debug output, logs or errors.
			fmt.Printf("Key:      %s\n", created.Key)
			fmt.Printf("Prefix:   %s\n", created.KeyPrefix)
			fmt.Printf("Agent:    %s\n", created.AgentID)
			fmt.Printf("Label:    %s\n", label)
			fmt.Printf("Created:  %s\n", created.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Expires:  %s\n", formatKeyExpiry(created.ExpiresAt, time.Now()))
			fmt.Println("Store this key now. It will not be shown in full again.")
			return nil
		},
	}

	cmd.Flags().String("agent", "", "Agent ID to scope the key to")
	cmd.Flags().String("label", "", "Optional label for the key")
	cmd.Flags().String("expires-in", "", "When the key stops working: 30d, 60d, 90d, 180d or 365d (default: never)")

	return cmd
}

func newKeysDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Revoke an agent-scoped API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")
			yes, _ := cmd.Flags().GetBool("yes")
			if !force && !yes {
				var confirm bool
				err := huh.NewConfirm().
					Title(fmt.Sprintf("Revoke key %s?", args[0])).
					Description("The key will stop working immediately. This action cannot be undone.").
					Value(&confirm).
					Run()
				if err != nil || !confirm {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			_, _, err = c.Delete("/v1/keys/" + args[0])
			if err != nil {
				return formatError(err)
			}

			fmt.Printf("Agent-scoped key %s revoked.\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (alias for --force)")

	return cmd
}

// keyExpiryOptions are the accepted --expires-in values (#593).
var keyExpiryOptions = []string{"30d", "60d", "90d", "180d", "365d"}

func validKeyExpiry(v string) bool {
	for _, o := range keyExpiryOptions {
		if v == o {
			return true
		}
	}
	return false
}

// formatKeyExpiry renders a key's expiry: "never", "expired", or the UTC time.
func formatKeyExpiry(expiresAt *time.Time, now time.Time) string {
	if expiresAt == nil {
		return "never"
	}
	if !expiresAt.After(now) {
		return "expired"
	}
	return expiresAt.UTC().Format("2006-01-02 15:04")
}
