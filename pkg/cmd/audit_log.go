package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

func newAuditLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit-log",
		Short: "Query the organization audit log",
		Long:  "List mutating API operations and auth lifecycle events for your organization, clamped to the per-tier retention window (Free: 3 days, Pro: 90 days, Custom: per contract).",
	}
	cmd.AddCommand(newAuditLogListCmd())
	return cmd
}

func newAuditLogListCmd() *cobra.Command {
	var (
		from           string
		to             string
		action         string
		actorKeyPrefix string
		actorType      string
		actorID        string
		requestID      string
		resource       string
		statusClass    string
		category       string
		cursor         string
		limit          int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List audit-log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			q := url.Values{}
			for key, value := range map[string]string{
				"from":             from,
				"to":               to,
				"action":           action,
				"actor_key_prefix": actorKeyPrefix,
				"actor_type":       actorType,
				"actor_id":         actorID,
				"request_id":       requestID,
				"resource":         resource,
				"status_class":     statusClass,
				"category":         category,
				"cursor":           cursor,
			} {
				if value != "" {
					q.Set(key, value)
				}
			}
			if limit > 0 {
				q.Set("limit", strconv.Itoa(limit))
			}

			path := "/v1/audit-log"
			if len(q) > 0 {
				path += "?" + q.Encode()
			}

			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			var resp client.AuditLogResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)

			if resp.RetentionDays != nil {
				fmt.Printf("Retention: last %d days", *resp.RetentionDays)
				if resp.RangeClamped {
					fmt.Print(" (requested range clamped to retention window)")
				}
				fmt.Println()
			} else {
				fmt.Println("Retention: per contract (Custom plan)")
			}

			if len(resp.Data) == 0 {
				fmt.Println("No entries found.")
				return nil
			}

			rows := make([][]string, 0, len(resp.Data))
			for _, e := range resp.Data {
				actor := "—"
				switch {
				case e.ActorLabel != nil && *e.ActorLabel != "":
					actor = *e.ActorLabel
				case e.ActorKeyPrefix != nil:
					actor = *e.ActorKeyPrefix
				}
				if e.ActorType != nil {
					actor += " [" + *e.ActorType + "]"
				}
				if e.AgentID != nil {
					actor += " (" + *e.AgentID + ")"
				}
				resource := "—"
				if e.Resource != nil && *e.Resource != "" {
					resource = *e.Resource
				}
				created := e.CreatedAt
				if len(created) >= 19 {
					created = created[:19] + "Z"
				}
				rows = append(rows, []string{
					created,
					e.Action,
					actor,
					resource,
					fmt.Sprintf("%d", e.Status),
				})
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"Time", "Action", "Actor", "Resource", "Status"},
				Rows:    rows,
			}, nc)

			if resp.Pagination.NextCursor != nil {
				fmt.Printf("\nMore results available. Use --cursor %q to continue.\n", *resp.Pagination.NextCursor)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Lower bound (ISO-8601 UTC, e.g. 2026-05-01T00:00:00Z)")
	cmd.Flags().StringVar(&to, "to", "", "Upper bound (ISO-8601 UTC). Defaults to now.")
	cmd.Flags().StringVar(&action, "action", "", "Filter by action (e.g. agent.create)")
	cmd.Flags().StringVar(&actorKeyPrefix, "actor", "", "Filter by actor key prefix (first 20 chars)")
	cmd.Flags().StringVar(&actorType, "actor-type", "", "Filter by who acted: session, org_key, agent_key or system")
	cmd.Flags().StringVar(&actorID, "actor-id", "", "Filter by actor id (an API key id, or the org id for console sessions)")
	cmd.Flags().StringVar(&requestID, "request-id", "", "Filter by the request id of the original call")
	cmd.Flags().StringVar(&resource, "resource", "", "Filter by an entity id or id prefix the request touched (e.g. cal_)")
	cmd.Flags().StringVar(&statusClass, "status-class", "", "Filter by response status class: 2xx, 3xx, 4xx or 5xx")
	cmd.Flags().StringVar(&category, "category", "", "Filter by action category (e.g. keys, calendars, billing, security)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor from a previous response")
	cmd.Flags().IntVar(&limit, "limit", 50, "Page size (1-200)")
	return cmd
}
