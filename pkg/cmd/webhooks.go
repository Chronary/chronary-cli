package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var validWebhookEvents = []string{
	"agent.created", "agent.updated",
	"event.created", "event.updated", "event.deleted", "event.started", "event.ended", "event.reminder",
	"event.hold_created", "event.hold_expired", "event.hold_released", "event.hold_confirmed",
	"proposal.created", "proposal.responded", "proposal.confirmed", "proposal.expired", "proposal.cancelled",
	"webhook.deactivated", "org.plan_changed",
	"calendar_connection.reauthorization_required",
	"usage.threshold_crossed",
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
	whCmd.AddCommand(newWebhooksDeliveryCmd())
	whCmd.AddCommand(newWebhooksTestCmd())
	whCmd.AddCommand(newWebhooksReplayCmd())
	whCmd.AddCommand(newWebhooksRotateSecretCmd())

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
				items, total, err := fetchAllPages(c, path, pageSizeWebhooks)
				if err != nil {
					return formatError(err)
				}
				body, err = rewrapList(items, total, pageSizeWebhooks)
				if err != nil {
					return fmt.Errorf("building response: %w", err)
				}
			} else {
				params := url.Values{}
				if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
					params.Set("limit", strconv.Itoa(v))
				}
				if v, _ := cmd.Flags().GetInt("offset"); v > 0 {
					params.Set("offset", strconv.Itoa(v))
				}
				path = appendQueryParams(path, params)
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

			payload, err := checkFileArg(args, 0)
			if err != nil {
				return err
			}
			if payload == nil {
				if err := requireFlagsSet(cmd, "url", "events"); err != nil {
					return err
				}
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
				// Intentional one-time display: the API returns the signing
				// secret only on create, and the receiver needs it to verify
				// deliveries. Printed once, only on stdout, never in logs or errors.
				fmt.Printf("Secret: %s\n", wh.Secret)
				fmt.Println("(Save this secret — it won't be shown again)")
			}
			return nil
		},
	}

	cmd.Flags().String("url", "", "Webhook endpoint URL (required unless using @file)")
	cmd.Flags().String("events", "", "Comma-separated event types (required unless using @file). Valid: "+strings.Join(validWebhookEvents, ", "))

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
			params := url.Values{}
			if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
				params.Set("limit", strconv.Itoa(v))
			}
			if v, _ := cmd.Flags().GetInt("offset"); v > 0 {
				params.Set("offset", strconv.Itoa(v))
			}
			if v, _ := cmd.Flags().GetString("status"); v != "" {
				params.Set("status", v)
			}
			if v, _ := cmd.Flags().GetBool("include-payload"); v {
				params.Set("include_payload", "true")
			}
			for flag, param := range map[string]string{
				"event-type":     "event_type",
				"created-after":  "created_after",
				"created-before": "created_before",
				"delivery-id":    "delivery_id",
				"cursor":         "cursor",
			} {
				if v, _ := cmd.Flags().GetString(flag); v != "" {
					params.Set(param, v)
				}
			}
			if cmd.Flags().Changed("test") {
				v, _ := cmd.Flags().GetBool("test")
				params.Set("is_test", strconv.FormatBool(v))
			}
			path = appendQueryParams(path, params)

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
			fmt.Printf("Success rate: %s  (delivered: %d  failed: %d  pending: %d  dropped: %d)\n\n",
				rateStr, stats.Delivered, stats.Failed, stats.Pending, stats.Dropped)

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
			if resp.NextCursor != nil {
				fmt.Printf("Next page: --cursor %s\n", *resp.NextCursor)
			}
			return nil
		},
	}

	cmd.Flags().Int("limit", 20, "Max results to return")
	cmd.Flags().Int("offset", 0, "Offset for pagination (cannot be combined with --cursor)")
	cmd.Flags().String("status", "", "Filter by status: pending, delivered, failed, dropped")
	cmd.Flags().Bool("include-payload", false, "Include full payload in output")
	cmd.Flags().String("event-type", "", "Filter by event type: exact (event.created) or a prefix ending in a dot (event.)")
	cmd.Flags().String("created-after", "", "Only deliveries created at or after this time (ISO 8601, UTC)")
	cmd.Flags().String("created-before", "", "Only deliveries created before this time (ISO 8601, UTC)")
	cmd.Flags().String("delivery-id", "", "Only the delivery with this id")
	cmd.Flags().Bool("test", false, "--test for test events only, --test=false to leave them out")
	cmd.Flags().String("cursor", "", "The next-page cursor printed by the previous page")

	return cmd
}

func newWebhooksDeliveryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delivery <webhook-id> <delivery-id>",
		Short: "Show one delivery and every attempt (status, duration, error)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Get("/v1/webhooks/" + args[0] + "/deliveries/" + args[1])
			if err != nil {
				return formatError(err)
			}

			var d client.WebhookDeliveryDetail
			if err := json.Unmarshal(body, &d); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, d) {
				return nil
			}

			nc := noColor(cmd)
			fmt.Printf("Delivery: %s\n", d.ID)
			fmt.Printf("Event:    %s\n", d.EventType)
			if d.ReplayOfDeliveryID != nil {
				fmt.Printf("Replay of: %s\n", *d.ReplayOfDeliveryID)
			}
			fmt.Printf("Status:   %s  (attempts: %d of %d)\n\n", output.ColorStatus(d.Status, nc), d.Attempts, d.MaxAttempts)

			rows := make([][]string, len(d.AttemptLog))
			for i, a := range d.AttemptLog {
				rows[i] = []string{
					strconv.Itoa(a.AttemptNumber),
					a.StartedAt.Format("2006-01-02 15:04:05"),
					intOrDash(a.ResponseStatus),
					msOrDash(a.DurationMs),
					stringOrDash(a.ErrorCode),
				}
			}
			output.RenderTable(output.TableDef{
				Headers: []string{"#", "Started", "Status", "Duration", "Error"},
				Rows:    rows,
			}, nc)
			return nil
		},
	}
	return cmd
}

func newWebhooksTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test <webhook-id>",
		Short: "Send a signed sample event to a webhook (one attempt, quota-exempt)",
		Long: "Send a signed sample payload of one event type to the endpoint. The delivery is " +
			"marked is_test, carries X-Chronary-Test: 1, makes one attempt and is sent even " +
			"when the endpoint is paused or disabled. Check the result with " +
			"`chronary webhooks delivery <webhook-id> <delivery-id>`. Limited to 10 per minute.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventType, _ := cmd.Flags().GetString("event")
			if eventType == "" {
				return fmt.Errorf("--event is required (for example --event event.created)")
			}
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Post("/v1/webhooks/"+args[0]+"/test", map[string]string{"event_type": eventType})
			if err != nil {
				return formatError(err)
			}

			var sent client.WebhookTestSent
			if err := json.Unmarshal(body, &sent); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, sent) {
				return nil
			}

			fmt.Printf("Sent test %s to webhook %s (delivery %s)\n", sent.EventType, sent.WebhookID, sent.DeliveryID)
			fmt.Printf("Check the result: chronary webhooks delivery %s %s\n", sent.WebhookID, sent.DeliveryID)
			return nil
		},
	}
	cmd.Flags().String("event", "", "Event type whose sample payload to send (the webhook must subscribe to it)")
	return cmd
}

func newWebhooksReplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replay <webhook-id> <delivery-id>",
		Short: "Send a failed or dropped delivery again as a new delivery (Developer or Pro)",
		Long: "Send a failed or dropped delivery again as a new delivery with its own id. The " +
			"stored payload goes out unchanged, signed with the endpoint's current secret, with an " +
			"X-Chronary-Replay-Of header naming the original. The replay follows the normal retry " +
			"schedule and counts toward the monthly delivery quota. The endpoint must be active. " +
			"Check the result with `chronary webhooks delivery <webhook-id> <new-delivery-id>`. " +
			"Requires a Developer or Pro plan. Limited to 30 per minute.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Post("/v1/webhooks/"+args[0]+"/deliveries/"+args[1]+"/replay", nil)
			if err != nil {
				return formatError(err)
			}

			var sent client.WebhookReplaySent
			if err := json.Unmarshal(body, &sent); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, sent) {
				return nil
			}

			fmt.Printf("Replayed %s as delivery %s (%s)\n", sent.ReplayOfDeliveryID, sent.DeliveryID, sent.EventType)
			fmt.Printf("Check the result: chronary webhooks delivery %s %s\n", sent.WebhookID, sent.DeliveryID)
			return nil
		},
	}
	return cmd
}

func newWebhooksRotateSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate-secret <webhook-id>",
		Short: "Rotate a webhook's signing secret, keeping the old one valid for a while (Developer or Pro)",
		Long: "Generate a new signing secret and print it once. Deliveries are signed with the new " +
			"secret in X-Signature at once; the old secret keeps signing X-Signature-Previous until " +
			"--expire-old-in has passed (now, 1h, 24h or 7d; default 24h), so your receiver can " +
			"switch secrets without rejecting deliveries. The SDK verifiers accept either header.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			expireOldIn, _ := cmd.Flags().GetString("expire-old-in")
			switch expireOldIn {
			case "now", "1h", "24h", "7d":
			default:
				return fmt.Errorf("--expire-old-in must be one of now, 1h, 24h, 7d")
			}
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Post("/v1/webhooks/"+args[0]+"/secret/rotate", map[string]string{"expire_old_in": expireOldIn})
			if err != nil {
				return formatError(err)
			}

			var rotated client.RotatedWebhookSecret
			if err := json.Unmarshal(body, &rotated); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, rotated) {
				return nil
			}

			fmt.Printf("Rotated the signing secret of webhook %s\n", rotated.WebhookID)
			// Intentional one-time display: the rotated secret is returned only
			// here. Printed once, only on stdout, never in logs or errors.
			fmt.Printf("New secret (shown once, store it now): %s\n", rotated.Secret)
			if rotated.PreviousSecretExpiresAt != nil {
				fmt.Printf("The old secret keeps working until %s (X-Signature-Previous)\n", *rotated.PreviousSecretExpiresAt)
			} else {
				fmt.Println("The old secret stopped working now")
			}
			return nil
		},
	}
	cmd.Flags().String("expire-old-in", "24h", "When the old secret stops working: now, 1h, 24h or 7d")
	return cmd
}

func intOrDash(v *int) string {
	if v == nil {
		return "—"
	}
	return strconv.Itoa(*v)
}

func msOrDash(v *int) string {
	if v == nil {
		return "—"
	}
	return strconv.Itoa(*v) + " ms"
}

func stringOrDash(v *string) string {
	if v == nil || *v == "" {
		return "—"
	}
	return *v
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
