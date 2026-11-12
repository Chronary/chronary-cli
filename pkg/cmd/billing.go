package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

// upgradeRequestPollInterval is how often `request-upgrade --wait` polls. The
// API guidance is "at most once a minute"; tests override this.
var upgradeRequestPollInterval = time.Minute

// upgradeRequestSleep is swapped out in tests.
var upgradeRequestSleep = time.Sleep

func newBillingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billing",
		Short: "Show plan and billing status; ask a human to upgrade",
		Long: `Show the account's plan and billing status.

Agents cannot pay, cancel, or change payment methods. When you hit a plan limit,
run "chronary billing request-upgrade" and give the approval URL to your human.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Get("/v1/billing")
			if err != nil {
				return formatError(err)
			}

			var summary client.BillingSummary
			if err := json.Unmarshal(body, &summary); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, summary) {
				return nil
			}

			renderBillingSummary(summary, noColor(cmd))
			return nil
		},
	}
	cmd.AddCommand(newBillingRequestUpgradeCmd(), newUpgradeRequestCmd())
	return cmd
}

func renderBillingSummary(s client.BillingSummary, nc bool) {
	rows := [][]string{
		{"Plan", s.Plan},
		{"Status", orDash(s.Status)},
		{"Interval", orDash(s.Interval)},
	}
	if s.Price != nil {
		rows = append(rows, []string{"Price", formatMoney(s.Price.Amount, s.Price.Currency) + "/" + s.Price.Interval + " (excl. tax)"})
	}
	if s.RenewsAt != nil {
		rows = append(rows, []string{"Renews", *s.RenewsAt})
	}
	if s.PendingPlan != nil && *s.PendingPlan != "" {
		pending := *s.PendingPlan
		if s.PendingPlanInterval != nil && *s.PendingPlanInterval != "" {
			pending += " (" + *s.PendingPlanInterval + "ly)"
		}
		if s.PendingPlanAt != nil {
			pending += " from " + *s.PendingPlanAt
		}
		rows = append(rows, []string{"Switching to", pending})
	}
	if s.CancelAt != nil {
		rows = append(rows, []string{"Cancels", *s.CancelAt})
	}
	if s.PlanExpiresAt != nil {
		rows = append(rows, []string{"Plan expires", *s.PlanExpiresAt})
	}
	rows = append(rows, []string{"Manage", s.ManageURL})
	output.RenderTable(output.TableDef{Headers: []string{"Field", "Value"}, Rows: rows}, nc)

	if len(s.UpgradeOptions) > 0 {
		fmt.Println("\nUpgrade options:")
		for _, o := range s.UpgradeOptions {
			line := fmt.Sprintf("  %s (per %s): %s", o.Plan, o.Interval, formatMoney(o.Amount, o.Currency))
			if o.SavingsVsMonthly > 0 {
				line += fmt.Sprintf(" (saves %s vs monthly)", formatMoney(o.SavingsVsMonthly, o.Currency))
			}
			fmt.Println(line)
		}
	}
	if s.UpgradeURL != nil {
		fmt.Printf("\nUpgrade: %s\n", *s.UpgradeURL)
	}
	if s.ContactURL != nil {
		fmt.Printf("Contact: %s\n", *s.ContactURL)
	}
	if r := s.PendingUpgradeRequest; r != nil {
		fmt.Printf("\nPending upgrade request %s (%s)\n", r.ID, r.Interval)
		if r.ApprovalURL != nil {
			fmt.Printf("Approval URL: %s\n", *r.ApprovalURL)
		}
	} else if s.SelfServeUpgrade {
		fmt.Println("\nRun `chronary billing request-upgrade` to ask your human to approve an upgrade.")
	}
}

func orDash(s *string) string {
	if s == nil || *s == "" {
		return "-"
	}
	return *s
}

// formatMoney renders minor units (2900) as "29.00 USD".
func formatMoney(amount int, currency string) string {
	return fmt.Sprintf("%d.%02d %s", amount/100, amount%100, strings.ToUpper(currency))
}

func newBillingRequestUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request-upgrade",
		Short: "Ask your human to upgrade your plan (prints an approval URL)",
		Long: `Create an upgrade request and print the approval URL. Give that URL to your
human; only a human can pay. If a request is already pending it is reused.

With --wait, poll once a minute until the request is completed, declined,
canceled or expired. After "completed", retry the call that was blocked about
15 seconds later.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			interval, _ := cmd.Flags().GetString("interval")
			reason, _ := cmd.Flags().GetString("reason")
			wait, _ := cmd.Flags().GetBool("wait")
			if interval != "month" && interval != "year" {
				return fmt.Errorf("--interval must be month or year")
			}

			payload := map[string]any{"interval": interval}
			if reason != "" {
				payload["reason"] = reason
			}
			body, _, err := c.Post("/v1/billing/upgrade-requests", payload)
			if err != nil {
				return formatError(err)
			}
			var req client.UpgradeRequest
			if err := json.Unmarshal(body, &req); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			structured := outputFormat(cmd) != "table"
			if !structured {
				printUpgradeRequest(req)
				if req.Reused {
					fmt.Println("A request is already pending; reusing it. Do not create another.")
				}
			}

			if wait {
				final, err := waitForUpgradeRequest(c, req, !structured)
				if err != nil {
					return err
				}
				req = final
				if !structured {
					fmt.Printf("\nUpgrade request %s is %s.\n", req.ID, req.Status)
					if req.Status == "completed" {
						fmt.Println("Retry the blocked call in about 15 seconds.")
					}
				}
			}

			printStructured(cmd, req)
			return nil
		},
	}
	cmd.Flags().String("interval", "month", "Billing interval: month or year")
	cmd.Flags().String("reason", "", "Why the upgrade is needed (max 500 chars, shown to your human)")
	cmd.Flags().Bool("wait", false, "Poll once a minute until the request reaches a final status")
	return cmd
}

func isTerminalUpgradeStatus(status string) bool {
	return status != "pending"
}

// waitForUpgradeRequest polls GET /v1/billing/upgrade-requests/{id} until the
// request leaves "pending". Polling honors the once-a-minute guidance.
func waitForUpgradeRequest(c *client.Client, req client.UpgradeRequest, announce bool) (client.UpgradeRequest, error) {
	if announce && !isTerminalUpgradeStatus(req.Status) {
		fmt.Printf("\nWaiting for approval (polling every %s)...\n", upgradeRequestPollInterval)
	}
	for !isTerminalUpgradeStatus(req.Status) {
		upgradeRequestSleep(upgradeRequestPollInterval)
		body, _, err := c.Get("/v1/billing/upgrade-requests/" + req.ID)
		if err != nil {
			return req, formatError(err)
		}
		if err := json.Unmarshal(body, &req); err != nil {
			return req, fmt.Errorf("parsing response: %w", err)
		}
	}
	return req, nil
}

func printUpgradeRequest(r client.UpgradeRequest) {
	fmt.Printf("Upgrade request: %s (%s, %s)\n", r.ID, r.Status, r.Interval)
	if r.ApprovalURL != nil {
		fmt.Printf("Approval URL:    %s\n", *r.ApprovalURL)
		fmt.Println("Give this URL to your human; only a human can pay.")
	}
	fmt.Printf("Expires:         %s\n", r.ExpiresAt)
}

func newUpgradeRequestCmd() *cobra.Command {
	root := &cobra.Command{Use: "upgrade-request", Short: "Inspect or cancel an upgrade request"}

	get := &cobra.Command{
		Use:   "get <id>",
		Short: "Show an upgrade request (poll at most once a minute)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}
			body, _, err := c.Get("/v1/billing/upgrade-requests/" + args[0])
			if err != nil {
				return formatError(err)
			}
			return renderUpgradeRequestResponse(cmd, body)
		},
	}

	cancel := &cobra.Command{
		Use:   "cancel <id>",
		Short: "Withdraw a pending upgrade request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}
			body, _, err := c.Post("/v1/billing/upgrade-requests/"+args[0]+"/cancel", nil)
			if err != nil {
				return formatError(err)
			}
			return renderUpgradeRequestResponse(cmd, body)
		},
	}

	root.AddCommand(get, cancel)
	return root
}

func renderUpgradeRequestResponse(cmd *cobra.Command, body []byte) error {
	var req client.UpgradeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}
	if printStructured(cmd, req) {
		return nil
	}
	printUpgradeRequest(req)
	return nil
}
