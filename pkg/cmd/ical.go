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

func newICalCmd() *cobra.Command {
	icalCmd := &cobra.Command{
		Use:     "ical",
		Aliases: []string{"ical-subscriptions"},
		Short:   "Manage iCal feed subscriptions",
	}

	icalCmd.AddCommand(newICalListCmd())
	icalCmd.AddCommand(newICalCreateCmd())
	icalCmd.AddCommand(newICalGetCmd())
	icalCmd.AddCommand(newICalUpdateCmd())
	icalCmd.AddCommand(newICalDeleteCmd())
	icalCmd.AddCommand(newICalSyncCmd())

	return icalCmd
}

func newICalListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List iCal subscriptions for an agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			agentID, _ := cmd.Flags().GetString("agent")
			if agentID == "" {
				return fmt.Errorf("--agent is required")
			}

			all, _ := cmd.Flags().GetBool("all")
			filterParams := []string{}
			if v, _ := cmd.Flags().GetString("status"); v != "" {
				filterParams = append(filterParams, "status="+v)
			}

			path := fmt.Sprintf("/v1/agents/%s/ical-subscriptions", agentID)
			if len(filterParams) > 0 {
				path += "?" + strings.Join(filterParams, "&")
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
				pagParams := []string{}
				if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
					pagParams = append(pagParams, fmt.Sprintf("limit=%d", v))
				}
				if v, _ := cmd.Flags().GetInt("offset"); v > 0 {
					pagParams = append(pagParams, fmt.Sprintf("offset=%d", v))
				}
				if len(pagParams) > 0 {
					sep := "?"
					if strings.Contains(path, "?") {
						sep = "&"
					}
					path += sep + strings.Join(pagParams, "&")
				}
				var fetchErr error
				body, _, fetchErr = c.Get(path)
				if fetchErr != nil {
					return formatError(fetchErr)
				}
			}

			var resp client.ListResponse[client.ICalSubscription]
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)
			rows := make([][]string, len(resp.Data))
			for i, sub := range resp.Data {
				label := ""
				if sub.Label != nil {
					label = *sub.Label
				}
				lastSync := "-"
				if sub.LastSyncedAt != nil {
					lastSync = (*sub.LastSyncedAt)[:19]
				}
				rows[i] = []string{
					sub.ID,
					sub.CalendarID,
					output.Truncate(label, 25),
					output.ColorStatus(sub.Status, nc),
					lastSync,
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "Calendar", "Label", "Status", "Last Synced"},
				Rows:    rows,
			}, nc)

			fmt.Printf("\nShowing %d of %d subscriptions\n", len(resp.Data), resp.Total)
			return nil
		},
	}

	cmd.Flags().String("agent", "", "Agent ID (required)")
	cmd.Flags().String("status", "", "Filter by status: active, error, paused")
	cmd.Flags().Int("limit", 50, "Max results to return")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().Bool("all", false, "Fetch all pages automatically")
	cmd.MarkFlagRequired("agent")

	return cmd
}

func newICalCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new iCal subscription",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			agentID, _ := cmd.Flags().GetString("agent")
			calendarID, _ := cmd.Flags().GetString("calendar")
			url, _ := cmd.Flags().GetString("url")
			label, _ := cmd.Flags().GetString("label")

			payload := map[string]any{
				"calendar_id": calendarID,
				"url":         url,
			}
			if label != "" {
				payload["label"] = label
			}

			path := fmt.Sprintf("/v1/agents/%s/ical-subscriptions", agentID)
			body, _, err := c.Post(path, payload)
			if err != nil {
				return formatError(err)
			}

			var sub client.ICalSubscription
			if err := json.Unmarshal(body, &sub); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, sub) {
				return nil
			}

			fmt.Printf("Created iCal subscription %s\n", sub.ID)
			fmt.Printf("Calendar: %s\n", sub.CalendarID)
			fmt.Printf("Status:   %s\n", sub.Status)
			return nil
		},
	}

	cmd.Flags().String("agent", "", "Agent ID (required)")
	cmd.Flags().String("calendar", "", "Calendar ID to sync into (required)")
	cmd.Flags().String("url", "", "iCal feed URL, must be HTTPS (required)")
	cmd.Flags().String("label", "", "Label for this subscription")
	cmd.MarkFlagRequired("agent")
	cmd.MarkFlagRequired("calendar")
	cmd.MarkFlagRequired("url")

	return cmd
}

func newICalGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an iCal subscription by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Get("/v1/ical-subscriptions/" + args[0])
			if err != nil {
				return formatError(err)
			}

			var sub client.ICalSubscription
			if err := json.Unmarshal(body, &sub); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, sub) {
				return nil
			}

			nc := noColor(cmd)
			label := ""
			if sub.Label != nil {
				label = *sub.Label
			}

			fmt.Printf("ID:          %s\n", sub.ID)
			fmt.Printf("Agent:       %s\n", sub.AgentID)
			fmt.Printf("Calendar:    %s\n", sub.CalendarID)
			fmt.Printf("Status:      %s\n", output.ColorStatus(sub.Status, nc))
			if label != "" {
				fmt.Printf("Label:       %s\n", label)
			}
			if sub.LastSyncedAt != nil {
				fmt.Printf("Last Synced: %s\n", *sub.LastSyncedAt)
			}
			if sub.LastError != nil {
				fmt.Printf("Last Error:  %s\n", *sub.LastError)
			}
			fmt.Printf("Created:     %s\n", sub.CreatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}

func newICalUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an iCal subscription",
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

			if cmd.Flags().Changed("label") {
				v, _ := cmd.Flags().GetString("label")
				payload["label"] = v
			}
			if cmd.Flags().Changed("url") {
				v, _ := cmd.Flags().GetString("url")
				payload["url"] = v
			}

			if len(payload) == 0 {
				return fmt.Errorf("at least one flag required: --label, --url")
			}

			body, _, err := c.Patch("/v1/ical-subscriptions/"+args[0], payload)
			if err != nil {
				return formatError(err)
			}

			var sub client.ICalSubscription
			if err := json.Unmarshal(body, &sub); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, sub) {
				return nil
			}

			fmt.Printf("Updated iCal subscription %s\n", sub.ID)
			return nil
		},
	}

	cmd.Flags().String("label", "", "New label")
	cmd.Flags().String("url", "", "New iCal feed URL (HTTPS)")

	return cmd
}

func newICalDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an iCal subscription",
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
					Title(fmt.Sprintf("Delete iCal subscription %s?", args[0])).
					Description("This will stop syncing events from this feed. This action cannot be undone.").
					Value(&confirm).
					Run()
				if err != nil || !confirm {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			_, _, err = c.Delete("/v1/ical-subscriptions/" + args[0])
			if err != nil {
				return formatError(err)
			}

			fmt.Printf("iCal subscription %s deleted.\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (alias for --force)")

	return cmd
}

func newICalSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync <id>",
		Short: "Trigger a manual sync for an iCal subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			_, _, err = c.Post("/v1/ical-subscriptions/"+args[0]+"/sync", nil)
			if err != nil {
				return formatError(err)
			}

			fmt.Printf("Sync triggered for iCal subscription %s.\n", args[0])
			return nil
		},
	}
}
