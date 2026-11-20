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
  - On a brand-new email: org_id, agent_id, and a restricted api_key. The key
    is restricted to the verify endpoint until you run ` + "`chronary auth verify`" + `.
  - On an email that already has an org: only an opaque message (so signup
    cannot be used to enumerate existing accounts).
  - While sign-ups are paused: ` + "`waitlisted: true`" + ` and a message, the same for
    every email, with no key.

With --anonymous (no --email): a claimable org with no email
(POST /v1/agent/sign-up/anonymous). The key works at once at Sandbox limits;
the org cannot create webhooks or verify. Give the printed claim URL to the
person who should own the calendar before it expires (72 hours): opening it
while signed in to the console moves everything into their account. Unclaimed,
the key stops working and the org and its data are deleted.

This command does NOT save credentials to a profile — copy the key it prints,
or pipe it into ` + "`chronary auth login --profile <name>`" + ` yourself.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")
			agentName, _ := cmd.Flags().GetString("agent-name")
			tosVersion, _ := cmd.Flags().GetString("tos-version")
			anonymous, _ := cmd.Flags().GetBool("anonymous")

			if anonymous {
				if email != "" {
					return fmt.Errorf("--anonymous signs up without an email: drop --email")
				}
				return runAnonymousSignup(cmd, agentName, tosVersion)
			}
			if !strings.Contains(email, "@") {
				return fmt.Errorf("--email must contain @ (or use --anonymous to sign up without an email)")
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
				OrgID   string `json:"org_id,omitempty"`
				AgentID string `json:"agent_id,omitempty"`
				APIKey  string `json:"api_key,omitempty"`
				// Set on the 202 sent while sign-ups are paused (#1655).
				Waitlisted bool   `json:"waitlisted,omitempty"`
				Message    string `json:"message"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			// Structured output carries the new key on purpose: agents parse
			// it from stdout, and it is never saved or shown again.
			if printStructured(cmd, resp) {
				return nil
			}

			if resp.Waitlisted {
				fmt.Println(resp.Message)
				return nil
			}

			if resp.APIKey == "" {
				fmt.Println(resp.Message)
				fmt.Println("(No new credentials returned — this email may already have an org.)")
				return nil
			}

			fmt.Printf("%s\n\n", resp.Message)
			fmt.Printf("Org ID:    %s\n", resp.OrgID)
			fmt.Printf("Agent ID:  %s\n", resp.AgentID)
			// Intentional one-time display of a newly issued key: signup does
			// not save it, so this stdout line is the caller's only copy. It is
			// printed once, only on stdout, and never in debug output, logs or
			// errors (the --debug response dump redacts it). The next-step hint
			// below points at this line instead of repeating the key.
			fmt.Printf("API key:   %s   (restricted — run `chronary auth verify` first)\n", resp.APIKey)
			fmt.Println()
			fmt.Println("Next: check your email for the OTP, then run:")
			fmt.Println("  chronary auth verify --otp <code> --api-key <the API key above>")
			return nil
		},
	}

	cmd.Flags().String("email", "", "Email to send the verification code to (required unless --anonymous)")
	cmd.Flags().String("agent-name", "", "Display name of the signing-up agent, 1-100 chars (required)")
	cmd.Flags().String("tos-version", "", "Exact ToS version string the caller has accepted (required)")
	cmd.Flags().Bool("anonymous", false, "Sign up without an email: a claimable org with a claim URL for its owner")
	cmd.MarkFlagRequired("agent-name")
	cmd.MarkFlagRequired("tos-version")

	return cmd
}

// runAnonymousSignup creates a claimable org with no email (#2335).
func runAnonymousSignup(cmd *cobra.Command, agentName, tosVersion string) error {
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

	body, _, err := c.Post("/v1/agent/sign-up/anonymous", map[string]any{
		"agent_name":  agentName,
		"tos_version": tosVersion,
		"source":      "agent",
	})
	if err != nil {
		return formatError(err)
	}

	var resp struct {
		OrgID     string `json:"org_id"`
		AgentID   string `json:"agent_id"`
		APIKey    string `json:"api_key"`
		ClaimURL  string `json:"claim_url"`
		ExpiresAt string `json:"expires_at"`
		Plan      string `json:"plan"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	// Structured output carries the key and the claim URL on purpose: agents
	// parse them from stdout, and neither is saved or shown again.
	if printStructured(cmd, resp) {
		return nil
	}

	fmt.Printf("Org ID:     %s\n", resp.OrgID)
	fmt.Printf("Agent ID:   %s\n", resp.AgentID)
	// One-time display of a newly issued key and claim link, as for signup
	// above: printed once, only on stdout, never in debug output or logs.
	fmt.Printf("API key:    %s   (works now, Sandbox limits)\n", resp.APIKey)
	fmt.Printf("Claim URL:  %s\n", resp.ClaimURL)
	fmt.Printf("Expires:    %s\n", resp.ExpiresAt)
	fmt.Println()
	fmt.Println("Give the claim URL to the person who should own this calendar before it expires.")
	fmt.Println("Unclaimed, the key stops working and the org and its data are deleted.")
	return nil
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
				Verified         bool    `json:"verified"`
				SandboxPromotion *string `json:"sandbox_promotion"`
				Message          string  `json:"message"`
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
