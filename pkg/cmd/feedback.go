package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var validFeedbackTypes = []string{"bug", "feature", "friction"}

func newFeedbackCmd() *cobra.Command {
	fbCmd := &cobra.Command{
		Use:     "feedback",
		Aliases: []string{"fb"},
		Short:   "Submit feedback to Chronary (bug, feature, or friction)",
	}

	fbCmd.AddCommand(newFeedbackSubmitCmd())

	return fbCmd
}

func newFeedbackSubmitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit structured feedback",
		Long: `Submit structured feedback about the API, SDK, or agent experience.

Rate-limited to 25 submissions per day per organization (UTC day).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			feedbackType, _ := cmd.Flags().GetString("type")
			message, _ := cmd.Flags().GetString("message")

			if !isValidFeedbackType(feedbackType) {
				return fmt.Errorf("invalid --type %q (must be one of: bug, feature, friction)", feedbackType)
			}
			if len(message) < 10 {
				return fmt.Errorf("--message must be at least 10 characters")
			}

			payload := map[string]any{
				"type":    feedbackType,
				"message": message,
			}

			body, _, err := c.Post("/v1/feedback", payload)
			if err != nil {
				return formatError(err)
			}

			var resp struct {
				Status string `json:"status"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			fmt.Printf("Feedback submitted: %s\n", resp.Status)
			return nil
		},
	}

	cmd.Flags().String("type", "", "Feedback type: bug, feature, or friction (required)")
	cmd.Flags().String("message", "", "Feedback message, 10-2000 characters (required)")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("message")

	return cmd
}

func isValidFeedbackType(t string) bool {
	for _, v := range validFeedbackTypes {
		if v == t {
			return true
		}
	}
	return false
}
