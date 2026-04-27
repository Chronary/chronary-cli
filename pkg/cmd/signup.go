package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newAuthSignupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signup",
		Short: "Create a new Chronary org via the agent signup API",
		Long: `Send the verification-code email and create a new org in one call.

The API returns:
  - On a brand-new email: org_id, agent_id, a restricted live api_key, and a
    fully-functional test_api_key. The live key is restricted to the verify
    endpoint until you run ` + "`chronary auth verify`" + `.
  - On an email that already has an org: only an opaque message (so signup
    cannot be used to enumerate existing accounts).

This command does NOT save credentials to a profile — copy the keys it prints,
or pipe them into ` + "`chronary auth login --profile <name>`" + ` yourself.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")
			agentName, _ := cmd.Flags().GetString("agent-name")
			tosVersion, _ := cmd.Flags().GetString("tos-version")

			if !strings.Contains(email, "@") {
				return fmt.Errorf("--email must contain @")
			}
			if agentName == "" {
				return fmt.Errorf("--agent-name is required")
			}
			if tosVersion == "" {
				return fmt.Errorf("--tos-version is required")
			}

			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"email":       email,
				"agent_name":  agentName,
				"tos_version": tosVersion,
			}

			body, _, err := c.Post("/v1/agent/sign-up", payload)
			if err != nil {
				return formatError(err)
			}

			var resp struct {
				OrgID      string `json:"org_id,omitempty"`
				AgentID    string `json:"agent_id,omitempty"`
				APIKey     string `json:"api_key,omitempty"`
				TestAPIKey string `json:"test_api_key,omitempty"`
				Message    string `json:"message"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			if resp.APIKey == "" {
				fmt.Println(resp.Message)
				fmt.Println("(No new credentials returned — this email may already have an org.)")
				return nil
			}

			fmt.Printf("%s\n\n", resp.Message)
			fmt.Printf("Org ID:        %s\n", resp.OrgID)
			fmt.Printf("Agent ID:      %s\n", resp.AgentID)
			fmt.Printf("Live API key:  %s   (restricted — run `chronary auth verify` first)\n", resp.APIKey)
			fmt.Printf("Test API key:  %s   (works immediately)\n", resp.TestAPIKey)
			fmt.Println()
			fmt.Println("Next: check your email for the OTP, then run:")
			fmt.Printf("  chronary auth verify --otp <code> --api-key %s\n", resp.APIKey)
			return nil
		},
	}

	cmd.Flags().String("email", "", "Email to send the verification code to (required)")
	cmd.Flags().String("agent-name", "", "Display name of the signing-up agent, 1-100 chars (required)")
	cmd.Flags().String("tos-version", "", "Exact ToS version string the caller has accepted (required)")
	cmd.MarkFlagRequired("email")
	cmd.MarkFlagRequired("agent-name")
	cmd.MarkFlagRequired("tos-version")

	return cmd
}

func newAuthVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Submit the OTP from signup to unlock full API access",
		Long: `Use the restricted live key returned by ` + "`chronary auth signup`" + ` to verify
the OTP that was emailed. After a successful verify, the same key unlocks
the full API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			otp, _ := cmd.Flags().GetString("otp")
			if len(otp) != 6 {
				return fmt.Errorf("--otp must be 6 digits")
			}

			body, _, err := c.Post("/v1/agent/verify", map[string]any{"otp": otp})
			if err != nil {
				return formatError(err)
			}

			var resp struct {
				Verified bool   `json:"verified"`
				Message  string `json:"message"`
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

	cmd.Flags().String("otp", "", "Six-digit code from the verification email (required)")
	cmd.MarkFlagRequired("otp")

	return cmd
}
