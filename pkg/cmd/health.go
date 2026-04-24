package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check API health status",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			body, _, err := c.Get("/health")
			latency := time.Since(start)

			if err != nil {
				return formatError(err)
			}

			var health client.HealthResponse
			if err := json.Unmarshal(body, &health); err != nil {
				return fmt.Errorf("unexpected response: %w", err)
			}

			format := outputFormat(cmd)
			nc := noColor(cmd)

			if format == "json" {
				output.PrintJSON(map[string]any{
					"status":  health.Status,
					"ts":      health.TS,
					"latency": latency.String(),
				})
				return nil
			}

			statusStr := output.ColorStatus(health.Status, nc)
			fmt.Printf("Status:  %s\n", statusStr)
			fmt.Printf("Time:    %s\n", health.TS)
			fmt.Printf("Latency: %s\n", latency.Round(time.Millisecond))
			return nil
		},
	}
}
