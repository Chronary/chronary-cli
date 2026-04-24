package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

func newCalendarsCmd() *cobra.Command {
	calendarsCmd := &cobra.Command{
		Use:     "calendars",
		Aliases: []string{"calendar", "cal"},
		Short:   "Manage calendars",
	}

	calendarsCmd.AddCommand(newCalendarsListCmd())
	calendarsCmd.AddCommand(newCalendarsCreateCmd())
	calendarsCmd.AddCommand(newCalendarsGetCmd())
	calendarsCmd.AddCommand(newCalendarsUpdateCmd())
	calendarsCmd.AddCommand(newCalendarsDeleteCmd())
	calendarsCmd.AddCommand(newCalendarsContextCmd())
	calendarsCmd.AddCommand(newCalendarsAvailabilityRulesCmd())

	return calendarsCmd
}

func newCalendarsContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "context <id>",
		Short: "Get the temporal context for a calendar (current/next/recent/upcoming events)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := fmt.Sprintf("/v1/calendars/%s/context", args[0])
			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			var ctx client.CalendarContext
			if err := json.Unmarshal(body, &ctx); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, ctx) {
				return nil
			}

			fmt.Printf("Calendar:     %s\n", ctx.CalendarID)
			fmt.Printf("Now:          %s\n", ctx.Now)
			fmt.Printf("Agent status: %s\n", ctx.AgentStatus)
			if ctx.CurrentEvent != nil {
				fmt.Printf("Current:      %s (%s → %s)\n", ctx.CurrentEvent.Title, ctx.CurrentEvent.StartTime, ctx.CurrentEvent.EndTime)
			}
			if ctx.NextEvent != nil {
				fmt.Printf("Next:         %s (%s)\n", ctx.NextEvent.Title, ctx.NextEvent.StartTime)
			}
			fmt.Printf("Recent:       %d events\n", len(ctx.RecentEvents))
			fmt.Printf("Upcoming:     %d events (next 24h)\n", len(ctx.Upcoming))
			return nil
		},
	}
}

func newCalendarsAvailabilityRulesCmd() *cobra.Command {
	rulesCmd := &cobra.Command{
		Use:     "availability-rules",
		Aliases: []string{"rules"},
		Short:   "Manage calendar availability rules (buffers, working hours)",
	}

	rulesCmd.AddCommand(&cobra.Command{
		Use:   "get <calendar_id>",
		Short: "Get availability rules for a calendar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := fmt.Sprintf("/v1/calendars/%s/availability-rules", args[0])
			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			var rules client.AvailabilityRules
			if err := json.Unmarshal(body, &rules); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, rules) {
				return nil
			}

			fmt.Printf("Calendar:        %s\n", rules.CalendarID)
			fmt.Printf("Buffer before:   %d min\n", rules.BufferBeforeMinutes)
			fmt.Printf("Buffer after:    %d min\n", rules.BufferAfterMinutes)
			fmt.Printf("Timezone:        %s\n", rules.Timezone)
			if len(rules.WorkingHours) > 0 {
				fmt.Printf("Working hours:   %s\n", string(rules.WorkingHours))
			}
			return nil
		},
	})

	rulesCmd.AddCommand(&cobra.Command{
		Use:   "set <calendar_id> <@file>",
		Short: "Set (upsert) availability rules for a calendar",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			payload, err := checkFileArg(args, 1)
			if err != nil {
				return err
			}
			if payload == nil {
				return fmt.Errorf("@file argument required (JSON body)")
			}

			path := fmt.Sprintf("/v1/calendars/%s/availability-rules", args[0])
			body, _, err := c.Put(path, payload)
			if err != nil {
				return formatError(err)
			}

			var rules client.AvailabilityRules
			if err := json.Unmarshal(body, &rules); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, rules) {
				return nil
			}

			fmt.Printf("Rules updated for calendar %s\n", rules.CalendarID)
			return nil
		},
	})

	deleteCmd := &cobra.Command{
		Use:   "delete <calendar_id>",
		Short: "Delete availability rules for a calendar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")
			yes, _ := cmd.Flags().GetBool("yes")
			if !force && !yes {
				var confirm bool
				err := huh.NewConfirm().
					Title(fmt.Sprintf("Delete availability rules for %s?", args[0])).
					Description("This action cannot be undone.").
					Value(&confirm).
					Run()
				if err != nil || !confirm {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			path := fmt.Sprintf("/v1/calendars/%s/availability-rules", args[0])
			_, _, err = c.Delete(path)
			if err != nil {
				return formatError(err)
			}

			fmt.Printf("Rules deleted for calendar %s.\n", args[0])
			return nil
		},
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	deleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (alias for --force)")
	rulesCmd.AddCommand(deleteCmd)

	return rulesCmd
}

func newCalendarsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List calendars",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			agentID, _ := cmd.Flags().GetString("agent")
			all, _ := cmd.Flags().GetBool("all")

			var path string
			if agentID != "" {
				path = fmt.Sprintf("/v1/agents/%s/calendars", agentID)
			} else {
				path = "/v1/calendars"
			}

			var body []byte
			if all {
				items, total, err := fetchAllPages(c, path, 200)
				if err != nil {
					return formatError(err)
				}
				body, err = rewrapList(items, total)
				if err != nil {
					return fmt.Errorf("building response: %w", err)
				}
			} else {
				params := []string{}
				if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
					params = append(params, fmt.Sprintf("limit=%d", v))
				}
				if v, _ := cmd.Flags().GetInt("offset"); v > 0 {
					params = append(params, fmt.Sprintf("offset=%d", v))
				}
				if len(params) > 0 {
					path += "?" + strings.Join(params, "&")
				}
				var fetchErr error
				body, _, fetchErr = c.Get(path)
				if fetchErr != nil {
					return formatError(fetchErr)
				}
			}

			var resp client.ListResponse[client.Calendar]
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)
			rows := make([][]string, len(resp.Data))
			for i, cal := range resp.Data {
				agentStr := ""
				if cal.AgentID != nil {
					agentStr = *cal.AgentID
				}
				rows[i] = []string{
					cal.ID,
					cal.Name,
					cal.Timezone,
					agentStr,
					cal.CreatedAt.Format("2006-01-02"),
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "Name", "Timezone", "Agent", "Created"},
				Rows:    rows,
			}, nc)

			fmt.Printf("\nShowing %d of %d calendars\n", len(resp.Data), resp.Total)
			return nil
		},
	}

	cmd.Flags().String("agent", "", "Filter by agent ID")
	cmd.Flags().Int("limit", 50, "Max results to return")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().Bool("all", false, "Fetch all pages automatically")

	return cmd
}

func newCalendarsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [@file]",
		Short: "Create a new calendar",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			agentID, _ := cmd.Flags().GetString("agent")

			payload, _ := checkFileArg(args, 0)
			if payload == nil {
				name, _ := cmd.Flags().GetString("name")
				timezone, _ := cmd.Flags().GetString("timezone")
				metadataStr, _ := cmd.Flags().GetString("metadata")

				payload = map[string]any{
					"name":     name,
					"timezone": timezone,
				}
				if metadataStr != "" {
					var meta map[string]any
					if err := json.Unmarshal([]byte(metadataStr), &meta); err != nil {
						return fmt.Errorf("--metadata must be valid JSON: %w", err)
					}
					payload["metadata"] = meta
				}
			}

			path := "/v1/calendars"
			if agentID != "" {
				path = fmt.Sprintf("/v1/agents/%s/calendars", agentID)
			}

			body, _, err := c.Post(path, payload)
			if err != nil {
				return formatError(err)
			}

			var cal client.Calendar
			if err := json.Unmarshal(body, &cal); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, cal) {
				return nil
			}

			fmt.Printf("Created calendar %s (%s)\n", cal.ID, cal.Name)
			return nil
		},
	}

	cmd.Flags().String("name", "", "Calendar name (required)")
	cmd.Flags().String("timezone", "UTC", "Calendar timezone (e.g., America/New_York)")
	cmd.Flags().String("agent", "", "Agent ID to own the calendar")
	cmd.Flags().String("metadata", "", "Calendar metadata as JSON string")
	cmd.MarkFlagRequired("name")

	return cmd
}

func newCalendarsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a calendar by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Get("/v1/calendars/" + args[0])
			if err != nil {
				return formatError(err)
			}

			var cal client.Calendar
			if err := json.Unmarshal(body, &cal); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, cal) {
				return nil
			}

			fmt.Printf("ID:       %s\n", cal.ID)
			fmt.Printf("Name:     %s\n", cal.Name)
			fmt.Printf("Timezone: %s\n", cal.Timezone)
			if cal.AgentID != nil {
				fmt.Printf("Agent:    %s\n", *cal.AgentID)
			}
			if cal.ICalURL != "" {
				fmt.Printf("iCal URL: %s\n", cal.ICalURL)
			}
			fmt.Printf("Created:  %s\n", cal.CreatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}

func newCalendarsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a calendar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			payload := map[string]any{}

			if cmd.Flags().Changed("name") {
				v, _ := cmd.Flags().GetString("name")
				payload["name"] = v
			}
			if cmd.Flags().Changed("timezone") {
				v, _ := cmd.Flags().GetString("timezone")
				payload["timezone"] = v
			}
			if cmd.Flags().Changed("metadata") {
				v, _ := cmd.Flags().GetString("metadata")
				var meta map[string]any
				if err := json.Unmarshal([]byte(v), &meta); err != nil {
					return fmt.Errorf("--metadata must be valid JSON: %w", err)
				}
				payload["metadata"] = meta
			}

			if len(payload) == 0 {
				return fmt.Errorf("at least one flag required: --name, --timezone, --metadata")
			}

			body, _, err := c.Patch("/v1/calendars/"+args[0], payload)
			if err != nil {
				return formatError(err)
			}

			var cal client.Calendar
			if err := json.Unmarshal(body, &cal); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, cal) {
				return nil
			}

			fmt.Printf("Updated calendar %s (%s)\n", cal.ID, cal.Name)
			return nil
		},
	}

	cmd.Flags().String("name", "", "New calendar name")
	cmd.Flags().String("timezone", "", "New timezone")
	cmd.Flags().String("metadata", "", "New metadata as JSON string")

	return cmd
}

func newCalendarsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a calendar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")
			yes, _ := cmd.Flags().GetBool("yes")
			if !force && !yes {
				var confirm bool
				err := huh.NewConfirm().
					Title(fmt.Sprintf("Delete calendar %s?", args[0])).
					Description("This will delete the calendar and all its events. This action cannot be undone.").
					Value(&confirm).
					Run()
				if err != nil || !confirm {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			_, _, err = c.Delete("/v1/calendars/" + args[0])
			if err != nil {
				return formatError(err)
			}

			fmt.Printf("Calendar %s deleted.\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (alias for --force)")

	return cmd
}
