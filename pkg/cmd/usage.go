package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

func newUsageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usage",
		Short: "Show current usage and limits",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Get("/v1/usage")
			if err != nil {
				return formatError(err)
			}

			var usage client.UsageResponse
			if err := json.Unmarshal(body, &usage); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, usage) {
				return nil
			}

			nc := noColor(cmd)

			fmt.Printf("Plan:   %s\n", usage.Plan)
			fmt.Printf("Period: %s to %s\n\n", usage.PeriodStart[:10], usage.PeriodEnd[:10])

			rows := [][]string{
				usageRow("Agents", usage.Agents, nc),
				usageRow("Calendars", usage.Calendars, nc),
				usageRow("Events", usage.Events, nc),
				usageRow("API Calls", usage.APICalls, nc),
				usageRow("Webhooks", usage.Webhooks, nc),
				usageRow("Availability Queries", usage.AvailabilityQueries, nc),
				usageRow("iCal Subscriptions", usage.ICalSubscriptions, nc),
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"Resource", "Used", "Limit", "Remaining"},
				Rows:    rows,
			}, nc)

			return nil
		},
	}
}

func usageRow(name string, counter client.UsageCounter, nc bool) []string {
	remaining := counter.Limit - counter.Used
	remainStr := fmt.Sprintf("%d", remaining)

	// Color the remaining count based on usage percentage
	if !nc && counter.Limit > 0 {
		pct := float64(counter.Used) / float64(counter.Limit)
		switch {
		case pct >= 0.9:
			remainStr = output.ColorStatus("error", nc)[:0] + output.ColorStatus(remainStr, nc)
		case pct >= 0.7:
			remainStr = output.ColorStatus(remainStr, nc)
		}
	}

	limitStr := fmt.Sprintf("%d", counter.Limit)
	if counter.Limit == 0 {
		limitStr = "unlimited"
		remainStr = "-"
	}

	return []string{
		name,
		fmt.Sprintf("%d", counter.Used),
		limitStr,
		remainStr,
	}
}
