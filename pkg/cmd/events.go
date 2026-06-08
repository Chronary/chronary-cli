package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

func newEventsCmd() *cobra.Command {
	eventsCmd := &cobra.Command{
		Use:     "events",
		Aliases: []string{"event"},
		Short:   "Manage events",
	}

	eventsCmd.AddCommand(newEventsListCmd())
	eventsCmd.AddCommand(newEventsCreateCmd())
	eventsCmd.AddCommand(newEventsGetCmd())
	eventsCmd.AddCommand(newEventsUpdateCmd())
	eventsCmd.AddCommand(newEventsDeleteCmd())
	eventsCmd.AddCommand(newEventsConfirmCmd())
	eventsCmd.AddCommand(newEventsReleaseCmd())

	return eventsCmd
}

func newEventsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List events",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			calendarID, _ := cmd.Flags().GetString("calendar")
			agentID, _ := cmd.Flags().GetString("agent")

			if calendarID == "" && agentID == "" {
				return fmt.Errorf("either --calendar or --agent is required")
			}

			var path string
			if agentID != "" {
				path = fmt.Sprintf("/v1/agents/%s/events", agentID)
			} else {
				path = fmt.Sprintf("/v1/calendars/%s/events", calendarID)
			}

			all, _ := cmd.Flags().GetBool("all")
			filterParams := url.Values{}
			if v, _ := cmd.Flags().GetString("start-after"); v != "" {
				filterParams.Set("start_after", v)
			}
			if v, _ := cmd.Flags().GetString("start-before"); v != "" {
				filterParams.Set("start_before", v)
			}
			if v, _ := cmd.Flags().GetString("status"); v != "" {
				filterParams.Set("status", v)
			}
			path = appendQueryParams(path, filterParams)

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
				pagParams := url.Values{}
				if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
					pagParams.Set("limit", strconv.Itoa(v))
				}
				if v, _ := cmd.Flags().GetInt("offset"); v > 0 {
					pagParams.Set("offset", strconv.Itoa(v))
				}
				path = appendQueryParams(path, pagParams)
				var fetchErr error
				body, _, fetchErr = c.Get(path)
				if fetchErr != nil {
					return formatError(fetchErr)
				}
			}

			var resp client.ListResponse[client.Event]
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)
			rows := make([][]string, len(resp.Data))
			for i, evt := range resp.Data {
				title := output.Truncate(evt.Title, 30)
				rows[i] = []string{
					evt.ID,
					title,
					evt.StartTime.Format("2006-01-02 15:04"),
					evt.EndTime.Format("2006-01-02 15:04"),
					output.ColorStatus(evt.Status, nc),
					evt.Source,
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "Title", "Start", "End", "Status", "Source"},
				Rows:    rows,
			}, nc)

			fmt.Printf("\nShowing %d of %d events\n", len(resp.Data), resp.Total)
			return nil
		},
	}

	cmd.Flags().String("calendar", "", "Calendar ID (required if --agent not set)")
	cmd.Flags().String("agent", "", "Agent ID (list all events across agent's calendars)")
	cmd.Flags().String("start-after", "", "Filter: start time after (ISO 8601)")
	cmd.Flags().String("start-before", "", "Filter: start time before (ISO 8601)")
	cmd.Flags().String("status", "", "Filter by status (confirmed, tentative, cancelled, hold)")
	cmd.Flags().Int("limit", 50, "Max results to return")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().Bool("all", false, "Fetch all pages automatically")

	return cmd
}

func newEventsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [@file]",
		Short: "Create a new event",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			calendarID, _ := cmd.Flags().GetString("calendar")
			if calendarID == "" {
				return fmt.Errorf("--calendar is required")
			}

			payload, _ := checkFileArg(args, 0)
			if payload == nil {
				title, _ := cmd.Flags().GetString("title")
				start, _ := cmd.Flags().GetString("start")
				end, _ := cmd.Flags().GetString("end")
				description, _ := cmd.Flags().GetString("description")
				allDay, _ := cmd.Flags().GetBool("all-day")
				status, _ := cmd.Flags().GetString("status")
				metadataStr, _ := cmd.Flags().GetString("metadata")
				holdExpiresAt, _ := cmd.Flags().GetString("hold-expires-at")
				holdPriority, _ := cmd.Flags().GetInt("hold-priority")

				payload = map[string]any{
					"title":      title,
					"start_time": start,
					"end_time":   end,
				}
				if description != "" {
					payload["description"] = description
				}
				if allDay {
					payload["all_day"] = true
				}
				if status != "" {
					payload["status"] = status
				}
				if cmd.Flags().Changed("reminders") {
					reminders, _ := cmd.Flags().GetIntSlice("reminders")
					payload["reminders"] = reminders
				}
				if metadataStr != "" {
					var meta map[string]any
					if err := json.Unmarshal([]byte(metadataStr), &meta); err != nil {
						return fmt.Errorf("--metadata must be valid JSON: %w", err)
					}
					payload["metadata"] = meta
				}
				if holdExpiresAt != "" {
					payload["hold_expires_at"] = holdExpiresAt
				}
				if cmd.Flags().Changed("hold-priority") {
					payload["hold_priority"] = holdPriority
				}
			}

			path := fmt.Sprintf("/v1/calendars/%s/events", calendarID)
			body, _, err := c.Post(path, payload)
			if err != nil {
				return formatError(err)
			}

			var evt client.Event
			if err := json.Unmarshal(body, &evt); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, evt) {
				return nil
			}

			fmt.Printf("Created event %s (%s)\n", evt.ID, evt.Title)
			return nil
		},
	}

	cmd.Flags().String("calendar", "", "Calendar ID (required)")
	cmd.Flags().String("title", "", "Event title (required)")
	cmd.Flags().String("start", "", "Start time, ISO 8601 (required)")
	cmd.Flags().String("end", "", "End time, ISO 8601 (required)")
	cmd.Flags().String("description", "", "Event description")
	cmd.Flags().Bool("all-day", false, "All-day event")
	cmd.Flags().String("status", "", "Event status: confirmed, tentative, cancelled, or hold")
	cmd.Flags().IntSlice("reminders", nil, "Reminder offsets in minutes before start (e.g. 10,1440). Max 5, each 1-40320. Omit to inherit calendar default; pass empty for none.")
	cmd.Flags().String("metadata", "", "Event metadata as JSON string")
	cmd.Flags().String("hold-expires-at", "", "Required when --status=hold. ISO 8601 timestamp 30s-15min in the future.")
	cmd.Flags().Int("hold-priority", 0, "Priority for hold conflict resolution (0-100). Higher-priority holds pre-empt lower.")
	cmd.MarkFlagRequired("calendar")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("start")
	cmd.MarkFlagRequired("end")

	return cmd
}

func newEventsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get an event by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			calendarID, _ := cmd.Flags().GetString("calendar")

			var path string
			if calendarID != "" {
				path = fmt.Sprintf("/v1/calendars/%s/events/%s", calendarID, args[0])
			} else {
				path = fmt.Sprintf("/v1/events/%s", args[0])
			}
			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			var evt client.Event
			if err := json.Unmarshal(body, &evt); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, evt) {
				return nil
			}

			nc := noColor(cmd)
			desc := ""
			if evt.Description != nil {
				desc = *evt.Description
			}

			fmt.Printf("ID:          %s\n", evt.ID)
			fmt.Printf("Calendar:    %s\n", evt.CalendarID)
			fmt.Printf("Title:       %s\n", evt.Title)
			fmt.Printf("Start:       %s\n", evt.StartTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("End:         %s\n", evt.EndTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("Status:      %s\n", output.ColorStatus(evt.Status, nc))
			fmt.Printf("All Day:     %v\n", evt.AllDay)
			fmt.Printf("Source:      %s\n", evt.Source)
			if desc != "" {
				fmt.Printf("Description: %s\n", desc)
			}
			fmt.Printf("Created:     %s\n", evt.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated:     %s\n", evt.UpdatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}

	cmd.Flags().String("calendar", "", "Calendar ID (optional — resolved from the event if omitted)")

	return cmd
}

func newEventsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			calendarID, _ := cmd.Flags().GetString("calendar")

			payload := map[string]any{}

			if cmd.Flags().Changed("title") {
				v, _ := cmd.Flags().GetString("title")
				payload["title"] = v
			}
			if cmd.Flags().Changed("start") {
				v, _ := cmd.Flags().GetString("start")
				payload["start_time"] = v
			}
			if cmd.Flags().Changed("end") {
				v, _ := cmd.Flags().GetString("end")
				payload["end_time"] = v
			}
			if cmd.Flags().Changed("description") {
				v, _ := cmd.Flags().GetString("description")
				payload["description"] = v
			}
			if cmd.Flags().Changed("status") {
				v, _ := cmd.Flags().GetString("status")
				payload["status"] = v
			}
			if cmd.Flags().Changed("reminders") {
				v, _ := cmd.Flags().GetIntSlice("reminders")
				payload["reminders"] = v
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
				return fmt.Errorf("at least one flag required: --title, --start, --end, --status, --reminders, --description, --metadata")
			}

			var path string
			if calendarID != "" {
				path = fmt.Sprintf("/v1/calendars/%s/events/%s", calendarID, args[0])
			} else {
				path = fmt.Sprintf("/v1/events/%s", args[0])
			}
			body, _, err := c.Patch(path, payload)
			if err != nil {
				return formatError(err)
			}

			var evt client.Event
			if err := json.Unmarshal(body, &evt); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, evt) {
				return nil
			}

			fmt.Printf("Updated event %s (%s)\n", evt.ID, evt.Title)
			return nil
		},
	}

	cmd.Flags().String("calendar", "", "Calendar ID (optional — resolved from the event if omitted)")
	cmd.Flags().String("title", "", "New title")
	cmd.Flags().String("start", "", "New start time (ISO 8601)")
	cmd.Flags().String("end", "", "New end time (ISO 8601)")
	cmd.Flags().String("description", "", "New description")
	cmd.Flags().String("status", "", "New status: confirmed, tentative, or cancelled")
	cmd.Flags().IntSlice("reminders", nil, "Reminder offsets in minutes before start (e.g. 10,1440). Max 5, each 1-40320. Pass empty to clear all reminders.")
	cmd.Flags().String("metadata", "", "New metadata as JSON string")

	return cmd
}

func newEventsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			calendarID, _ := cmd.Flags().GetString("calendar")

			force, _ := cmd.Flags().GetBool("force")
			yes, _ := cmd.Flags().GetBool("yes")
			if !force && !yes {
				var confirm bool
				err := huh.NewConfirm().
					Title(fmt.Sprintf("Delete event %s?", args[0])).
					Description("This action cannot be undone.").
					Value(&confirm).
					Run()
				if err != nil || !confirm {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			var path string
			if calendarID != "" {
				path = fmt.Sprintf("/v1/calendars/%s/events/%s", calendarID, args[0])
			} else {
				path = fmt.Sprintf("/v1/events/%s", args[0])
			}
			_, _, err = c.Delete(path)
			if err != nil {
				return formatError(err)
			}

			fmt.Printf("Event %s deleted.\n", args[0])
			return nil
		},
	}

	cmd.Flags().String("calendar", "", "Calendar ID (optional — resolved from the event if omitted)")
	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (alias for --force)")

	return cmd
}

func newEventsConfirmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "confirm <id>",
		Short: "Confirm a held event (status=hold → confirmed)",
		Long:  "Promotes a tentative hold to a confirmed event. The hold must still be active (not past its hold_expires_at).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := fmt.Sprintf("/v1/events/%s/confirm", args[0])
			body, _, err := c.Put(path, nil)
			if err != nil {
				return formatError(err)
			}

			var evt client.Event
			if err := json.Unmarshal(body, &evt); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, evt) {
				return nil
			}

			fmt.Printf("Confirmed event %s (%s)\n", evt.ID, evt.Title)
			return nil
		},
	}
}

func newEventsReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release <id>",
		Short: "Release a held event (soft-deletes the hold)",
		Long:  "Manually releases a hold before its TTL, freeing the slot for other agents.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := fmt.Sprintf("/v1/events/%s/release", args[0])
			body, _, err := c.Put(path, nil)
			if err != nil {
				return formatError(err)
			}

			var evt client.Event
			if err := json.Unmarshal(body, &evt); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, evt) {
				return nil
			}

			fmt.Printf("Released hold %s\n", evt.ID)
			return nil
		},
	}
}
