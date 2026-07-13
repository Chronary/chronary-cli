package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

func newAvailabilityCmd() *cobra.Command {
	availCmd := &cobra.Command{
		Use:     "availability",
		Aliases: []string{"avail"},
		Short:   "Query availability",
	}

	availCmd.AddCommand(newAvailAgentCmd())
	availCmd.AddCommand(newAvailCalendarCmd())
	availCmd.AddCommand(newAvailCrossCmd())

	return availCmd
}

func availabilityFlags(cmd *cobra.Command) {
	cmd.Flags().String("start", "", "Start of window, ISO 8601 (required)")
	cmd.Flags().String("end", "", "End of window, ISO 8601 (required)")
	cmd.Flags().String("slot-duration", "30m", "Slot duration: 15m, 30m, 45m, 1h, 2h")
	cmd.Flags().Bool("include-busy", false, "Include busy blocks in response")
	cmd.Flags().Bool("allow-stale", false, "Allow usable stale human-calendar cache (exploration only; default is fail closed)")
	cmd.MarkFlagRequired("start")
	cmd.MarkFlagRequired("end")
}

func buildAvailParams(cmd *cobra.Command) url.Values {
	params := url.Values{}
	if v, _ := cmd.Flags().GetString("start"); v != "" {
		params.Set("start", v)
	}
	if v, _ := cmd.Flags().GetString("end"); v != "" {
		params.Set("end", v)
	}
	if v, _ := cmd.Flags().GetString("slot-duration"); v != "" {
		params.Set("slot_duration", v)
	}
	if v, _ := cmd.Flags().GetBool("include-busy"); v {
		params.Set("include_busy", "true")
	}
	if v, _ := cmd.Flags().GetBool("allow-stale"); v {
		params.Set("allow_stale", "true")
	}
	return params
}

func printAvailability(cmd *cobra.Command, body []byte) error {
	var resp client.AvailabilityResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if printStructured(cmd, resp) {
		return nil
	}

	nc := noColor(cmd)
	if resp.AvailabilityState != "" && resp.AvailabilityState != "complete" {
		fmt.Printf("Warning: availability is %s; required human-calendar data is not current.\n", resp.AvailabilityState)
	}

	if len(resp.Slots) == 0 {
		fmt.Println("No available slots found.")
		return nil
	}

	rows := make([][]string, len(resp.Slots))
	for i, slot := range resp.Slots {
		rows[i] = []string{slot.Start, slot.End}
	}

	output.RenderTable(output.TableDef{
		Headers: []string{"Start", "End"},
		Rows:    rows,
	}, nc)

	fmt.Printf("\n%d available slots\n", len(resp.Slots))

	if len(resp.Busy) > 0 {
		fmt.Printf("\n%d busy blocks:\n", len(resp.Busy))
		busyRows := make([][]string, len(resp.Busy))
		for i, b := range resp.Busy {
			title := ""
			if b.Title != nil {
				title = *b.Title
			}
			busyRows[i] = []string{b.Start, b.End, title}
		}
		output.RenderTable(output.TableDef{
			Headers: []string{"Start", "End", "Title"},
			Rows:    busyRows,
		}, nc)
	}

	return nil
}

func newAvailAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent <id>",
		Short: "Get availability for an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := fmt.Sprintf("/v1/agents/%s/availability", args[0])
			path = appendQueryParams(path, buildAvailParams(cmd))
			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			return printAvailability(cmd, body)
		},
	}

	availabilityFlags(cmd)
	return cmd
}

func newAvailCalendarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendar <id>",
		Short: "Get availability for a calendar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := fmt.Sprintf("/v1/calendars/%s/availability", args[0])
			path = appendQueryParams(path, buildAvailParams(cmd))
			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			return printAvailability(cmd, body)
		},
	}

	availabilityFlags(cmd)
	return cmd
}

func newAvailCrossCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cross",
		Short: "Get cross-agent availability (shared free slots)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			agents, _ := cmd.Flags().GetString("agents")
			if agents == "" {
				return fmt.Errorf("--agents is required (comma-separated agent IDs)")
			}

			params := buildAvailParams(cmd)
			params.Set("agents", agents)

			if v, _ := cmd.Flags().GetString("calendars"); v != "" {
				params.Set("calendars", v)
			}

			path := "/v1/availability"
			path = appendQueryParams(path, params)
			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			return printAvailability(cmd, body)
		},
	}

	availabilityFlags(cmd)
	cmd.Flags().String("agents", "", "Comma-separated agent IDs (required)")
	cmd.Flags().String("calendars", "", "Comma-separated calendar IDs (optional filter)")
	cmd.MarkFlagRequired("agents")

	return cmd
}
