package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// newAuthWaitlistCmd wires `chronary auth waitlist` (#442). It POSTs to
// /v1/waitlist — public endpoint, no API key required. Useful for soft-launch
// when open signup is gated. The API answers every email with the same body
// (#1655), so there is nothing to print beyond its message.
func newAuthWaitlistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "waitlist",
		Short: "Join the Chronary waitlist (private preview)",
		Long: `Join the Chronary waitlist during private preview.

Adds the email to the waitlist. No API keys are issued; when a founder
approves the request, an invite is emailed, and accepting it creates the
account.

The response is the same whether the email is new, already on the list, or
already has an account, so re-running is safe. Only a new email is sent the
confirmation email.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")
			name, _ := cmd.Flags().GetString("name")
			tosVersion, _ := cmd.Flags().GetString("tos-version")
			intent, _ := cmd.Flags().GetString("intent")

			if !strings.Contains(email, "@") {
				return fmt.Errorf("--email must contain @")
			}

			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"email": email,
			}
			if name != "" {
				payload["name"] = name
			}
			if tosVersion != "" {
				payload["tos_version"] = tosVersion
			}
			if intent != "" {
				payload["intent"] = intent
			}

			body, _, err := c.Post("/v1/waitlist", payload)
			if err != nil {
				return formatError(err)
			}

			var resp struct {
				Waitlisted bool   `json:"waitlisted"`
				Message    string `json:"message"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			fmt.Println(resp.Message)
			return nil
		},
	}

	cmd.Flags().String("email", "", "Email to enroll on the waitlist (required)")
	cmd.Flags().String("name", "", "Optional display name (defaults to the email's local-part)")
	cmd.Flags().String("tos-version", "", "Optional ToS version string (recommended for compliance)")
	cmd.Flags().String("intent", "", "Optional: what you are building (up to 200 characters)")
	cmd.MarkFlagRequired("email")

	return cmd
}
