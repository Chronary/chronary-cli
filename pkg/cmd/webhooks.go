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

var validWebhookEvents = []string{
	"agent.created", "agent.updated",
	"event.created", "event.updated", "event.deleted", "event.started", "event.ended",
	"event.hold_created", "event.hold_expired", "event.hold_released", "event.hold_confirmed",
	"proposal.created", "proposal.responded", "proposal.confirmed", "proposal.expired", "proposal.cancelled",
}

func newWebhooksCmd() *cobra.Command {
	whCmd := &cobra.Command{
		Use:     "webhooks",
		Aliases: []string{"webhook", "wh"},
		Short:   "Manage webhook subscriptions",
	}

	whCmd.AddCommand(newWebhooksListCmd())
	whCmd.AddCommand(newWebhooksCreateCmd())
	whCmd.AddCommand(newWebhooksGetCmd())
	whCmd.AddCommand(newWebhooksUpdateCmd())
	whCmd.AddCommand(newWebhooksDeleteCmd())
	whCmd.AddCommand(newWebhooksDeliveriesCmd())

	return whCmd
}

func newWebhooksListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhook subscriptions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			all, _ := cmd.Flags().GetBool("all")
			path := "/v1/webhooks"

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

			var resp client.ListResponse[client.Webhook]
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)
			rows := make([][]string, len(resp.Data))
			for i, wh := range resp.Data {
				activeStr := "active"
				if !wh.Active {
					activeStr = "inactive"
				}
				rows[i] = []string{
					wh.ID,
					output.Truncate(wh.URL, 40),
					strings.Join(wh.Events, ", "),
					output.ColorStatus(activeStr, nc),
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "URL", "Events", "Status"},
				Rows:    rows,
			}, nc)

			fmt.Printf("\nShowing %d of %d webhooks\n", len(resp.Data), resp.Total)
			return nil
		},
	}

	cmd.Flags().Int("limit", 20, "Max results to return")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().Bool("all", false, "Fetch all pages automatically")

	return cmd
}

func newWebhooksCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [@file]",
		Short: "Create a new webhook subscription",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			payload, _ := checkFileArg(args, 0)
			if payload == nil {
				url, _ := cmd.Flags().GetString("url")
				eventsStr, _ := cmd.Flags().GetString("events")

				events := strings.Split(eventsStr, ",")
				for i := range events {
					events[i] = strings.TrimSpace(events[i])
				}

				payload = map[string]any{
					"url":    url,
					"events": events,
				}
			}

			body, _, err := c.Post("/v1/webhooks", payload)
			if err != nil {
				return formatError(err)
			}

			var wh client.Webhook
			if err := json.Unmarshal(body, &wh); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, wh) {
				return nil
			}

			fmt.Printf("Created webhook %s\n", wh.ID)
			fmt.Printf("URL:    %s\n", wh.URL)
			fmt.Printf("Events: %s\n", strings.Join(wh.Events, ", "))
			if wh.Secret != "" {
				fmt.Printf("Secret: %s\n", wh.Secret)
				fmt.Println("(Save this secret — it won't be shown again)")
			}
			return nil
		},
	}

	cmd.Flags().String("url", "", "Webhook endpoint URL (required)")
	cmd.Flags().String("events", "", "Comma-separated event types (required). Valid: "+strings.Join(validWebhookEvents, ", "))
	cmd.MarkFlagRequired("url")
	cmd.MarkFlagRequired("events")

	return cmd
}

func newWebhooksGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a webhook subscription by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Get("/v1/webhooks/" + args[0])
			if err != nil {
				return formatError(err)
			}

			var wh client.Webhook
			if err := json.Unmarshal(body, &wh); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, wh) {
				return nil
			}

			nc := noColor(cmd)
			activeStr := "active"
			if !wh.Active {
				activeStr = "inactive"
			}

			fmt.Printf("ID:      %s\n", wh.ID)
			fmt.Printf("URL:     %s\n", wh.URL)
			fmt.Printf("Events:  %s\n", strings.Join(wh.Events, ", "))
			fmt.Printf("Status:  %s\n", output.ColorStatus(activeStr, nc))
			fmt.Printf("Created: %s\n", wh.CreatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}

func newWebhooksUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a webhook subscription",
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

			if cmd.Flags().Changed("url") {
				v, _ := cmd.Flags().GetString("url")
				payload["url"] = v
			}
			if cmd.Flags().Changed("events") {
				v, _ := cmd.Flags().GetString("events")
				events := strings.Split(v, ",")
				for i := range events {
					events[i] = strings.TrimSpace(events[i])
				}
				payload["events"] = events
			}
			if cmd.Flags().Changed("active") {
				v, _ := cmd.Flags().GetBool("active")
				payload["active"] = v
			}

			if len(payload) == 0 {
				return fmt.Errorf("at least one flag required: --url, --events, --active")
			}

			body, _, err := c.Patch("/v1/webhooks/"+args[0], payload)
			if err != nil {
				return formatError(err)
			}

			var wh client.Webhook
			if err := json.Unmarshal(body, &wh); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, wh) {
				return nil
			}

			fmt.Printf("Updated webhook %s\n", wh.ID)
			return nil
		},
	}

	cmd.Flags().String("url", "", "New endpoint URL")
	cmd.Flags().String("events", "", "New comma-separated event types")
	cmd.Flags().Bool("active", true, "Enable or disable the webhook")

	return cmd
}

func newWebhooksDeliveriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deliveries <webhook-id>",
		Short: "List delivery history for a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			webhookID := args[0]
			path := "/v1/webhooks/" + webhookID + "/deliveries"
			params := []string{}
			if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
				params = append(params, fmt.Sprintf("limit=%d", v))
			}
			if v, _ := cmd.Flags().GetInt("offset"); v > 0 {
				params = append(params, fmt.Sprintf("offset=%d", v))
			}
			if v, _ := cmd.Flags().GetString("status"); v != "" {
				params = append(params, "status="+v)
			}
			if v, _ := cmd.Flags().GetBool("include-payload"); v {
				params = append(params, "include_payload=true")
			}
			if len(params) > 0 {
				path += "?" + strings.Join(params, "&")
			}

			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			var resp client.WebhookDeliveryListResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)
			stats := resp.Stats
			successTotal := stats.Delivered + stats.Failed
			var rateStr string
			if successTotal > 0 {
				rate := float64(stats.Delivered) / float64(successTotal) * 100
				rateStr = fmt.Sprintf("%.1f%%", rate)
			} else {
				rateStr = "—"
			}
			fmt.Printf("Webhook: %s\n", webhookID)
			fmt.Printf("Success rate: %s  (delivered: %d  failed: %d  pending: %d)\n\n",
				rateStr, stats.Delivered, stats.Failed, stats.Pending)

			rows := make([][]string, len(resp.Data))
			for i, d := range resp.Data {
				lastAt := "—"
				if d.LastAttemptAt != nil {
					lastAt = d.LastAttemptAt.Format("2006-01-02 15:04:05")
				}
				rows[i] = []string{
					d.ID,
					d.EventType,
					output.ColorStatus(d.Status, nc),
					fmt.Sprintf("%d", d.Attempts),
					lastAt,
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "Event Type", "Status", "Attempts", "Last Attempt"},
				Rows:    rows,
			}, nc)

			fmt.Printf("\nShowing %d of %d deliveries\n", len(resp.Data), resp.Total)
			return nil
		},
	}

	cmd.Flags().Int("limit", 20, "Max results to return")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().String("status", "", "Filter by status: pending, delivered, failed")
	cmd.Flags().Bool("include-payload", false, "Include full payload in output")

	return cmd
}

func newWebhooksDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook subscription",
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
					Title(fmt.Sprintf("Delete webhook %s?", args[0])).
					Description("This action cannot be undone.").
					Value(&confirm).
					Run()
				if err != nil || !confirm {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			_, _, err = c.Delete("/v1/webhooks/" + args[0])
			if err != nil {
				return formatError(err)
			}

			fmt.Printf("Webhook %s deleted.\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (alias for --force)")

	return cmd
}
