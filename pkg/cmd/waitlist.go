package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// newAuthWaitlistCmd wires `chronary auth waitlist` (#442). It POSTs to
// /v1/waitlist — public endpoint, no API key required. Useful for soft-launch
// when open signup is gated; an admin flips the org to active later.
func newAuthWaitlistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "waitlist",
		Short: "Join the Chronary waitlist (private preview)",
		Long: `Join the Chronary waitlist during private preview.

Creates a real org row flagged as ` + "`is_waitlisted: true`" + `. No API keys are
issued; an admin will flip the flag to activate, after which the holder can
sign in normally and run ` + "`chronary auth login`" + `.

Idempotent: re-running with the same email returns the existing waitlist row
instead of erroring. An active (non-waitlisted) account at the same email
returns a 409 — sign in instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")
			name, _ := cmd.Flags().GetString("name")
			tosVersion, _ := cmd.Flags().GetString("tos-version")

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

			body, _, err := c.Post("/v1/waitlist", payload)
			if err != nil {
				return formatError(err)
			}

			var resp struct {
				Data struct {
					ID            string `json:"id"`
					Email         string `json:"email"`
					IsWaitlisted  bool   `json:"is_waitlisted"`
					WaitlistedAt  string `json:"waitlisted_at"`
					SignupSource  string `json:"signup_source"`
				} `json:"data"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			fmt.Println(resp.Message)
			fmt.Printf("Org ID:         %s\n", resp.Data.ID)
			fmt.Printf("Email:          %s\n", resp.Data.Email)
			fmt.Printf("Waitlisted at:  %s\n", resp.Data.WaitlistedAt)
			return nil
		},
	}

	cmd.Flags().String("email", "", "Email to enroll on the waitlist (required)")
	cmd.Flags().String("name", "", "Optional display name (defaults to the email's local-part)")
	cmd.Flags().String("tos-version", "", "Optional ToS version string (recommended for compliance)")
	cmd.MarkFlagRequired("email")

	return cmd
}
