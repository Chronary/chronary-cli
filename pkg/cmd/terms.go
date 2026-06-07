package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newTermsCmd() *cobra.Command {
	termsCmd := &cobra.Command{
		Use:   "terms",
		Short: "Manage terms-of-service acceptance",
	}

	termsCmd.AddCommand(newTermsAcceptCmd())

	return termsCmd
}

func newTermsAcceptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accept",
		Short: "Re-accept the current Chronary terms of service",
		Long: `Re-accept the current Chronary terms of service for the calling org.

Use this when a response carries the Chronary-Terms-Upgrade-Required header
after a material ToS bump — it clears the upgrade requirement for Bearer-key
clients that have no console session.

Requires an org-level API key (chr_sk_*). Agent-scoped keys cannot accept
org-wide terms (403). A stale --tos-version returns 409 tos_version_stale.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			tosVersion, _ := cmd.Flags().GetString("tos-version")
			if tosVersion == "" {
				return fmt.Errorf("--tos-version is required")
			}

			payload := map[string]any{
				"tos_version": tosVersion,
			}

			body, _, err := c.Post("/v1/terms/accept", payload)
			if err != nil {
				return formatError(err)
			}

			var resp struct {
				AcceptedTermsVersion string `json:"accepted_terms_version"`
				AcceptedTermsAt      string `json:"accepted_terms_at"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			fmt.Printf("Accepted terms version %s\n", resp.AcceptedTermsVersion)
			fmt.Printf("Accepted at:           %s\n", resp.AcceptedTermsAt)
			return nil
		},
	}

	cmd.Flags().String("tos-version", "", "Exact current ToS version string to accept (required)")
	cmd.MarkFlagRequired("tos-version")

	return cmd
}
