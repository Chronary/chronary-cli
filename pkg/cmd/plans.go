package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

func newPlansCmd() *cobra.Command {
	plansCmd := &cobra.Command{
		Use:   "plans",
		Short: "Browse the public Chronary plan catalog",
	}
	plansCmd.AddCommand(newPlansListCmd())
	return plansCmd
}

func newPlansListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available plans with prices and limits",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}

			// /v1/plans is public — no API key required.
			body, _, err := c.Get("/v1/plans")
			if err != nil {
				return formatError(err)
			}

			var resp client.PlansListResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)

			rows := make([][]string, 0, len(resp.Plans))
			for _, p := range resp.Plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					formatPlanPrice(p),
					formatAPICallsLimit(p),
					formatRecommended(p.Recommended),
				})
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "Name", "Price", "API Calls / mo", "Recommended"},
				Rows:    rows,
			}, nc)

			return nil
		},
	}
}

// formatPlanPrice renders cents + currency as "$29.00/mo" or "Contact sales" for custom-priced tiers.
func formatPlanPrice(p client.Plan) string {
	if p.CustomPricing || p.Price == nil || p.Currency == nil {
		return "Contact sales"
	}
	cents := *p.Price
	if cents == 0 {
		return "Free"
	}
	return fmt.Sprintf("$%d.%02d/mo %s", cents/100, cents%100, *p.Currency)
}

func formatAPICallsLimit(p client.Plan) string {
	if p.Limits == nil || p.Limits.APICalls == nil {
		return "—"
	}
	return fmt.Sprintf("%d", *p.Limits.APICalls)
}

func formatRecommended(recommended bool) string {
	if recommended {
		return "yes"
	}
	return ""
}
