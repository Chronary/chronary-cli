package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

func newSchedulingCmd() *cobra.Command {
	schedCmd := &cobra.Command{
		Use:     "scheduling",
		Aliases: []string{"schedule", "proposals"},
		Short:   "Manage scheduling proposals",
	}

	schedCmd.AddCommand(newSchedulingCreateCmd())
	schedCmd.AddCommand(newSchedulingListCmd())
	schedCmd.AddCommand(newSchedulingGetCmd())
	schedCmd.AddCommand(newSchedulingRespondCmd())
	schedCmd.AddCommand(newSchedulingResolveCmd())
	schedCmd.AddCommand(newSchedulingCancelCmd())

	return schedCmd
}

func newSchedulingCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <@file>",
		Short: "Create a new scheduling proposal",
		Args:  cobra.ExactArgs(1),
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
				return fmt.Errorf("@file argument required (JSON body)")
			}

			body, _, err := c.Post("/v1/scheduling/proposals", payload)
			if err != nil {
				return formatError(err)
			}

			var proposal client.Proposal
			if err := json.Unmarshal(body, &proposal); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, proposal) {
				return nil
			}

			fmt.Printf("Created proposal %s\n", proposal.ID)
			fmt.Printf("Title:    %s\n", proposal.Title)
			fmt.Printf("Status:   %s\n", proposal.Status)
			fmt.Printf("Calendar: %s\n", proposal.CalendarID)
			return nil
		},
	}
	return cmd
}

func newSchedulingListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List scheduling proposals",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			all, _ := cmd.Flags().GetBool("all")
			path := "/v1/scheduling/proposals"

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
				params := url.Values{}
				if v, _ := cmd.Flags().GetString("status"); v != "" {
					params.Set("status", v)
				}
				if v, _ := cmd.Flags().GetString("organizer"); v != "" {
					params.Set("organizer_agent_id", v)
				}
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

			var resp client.ListResponse[client.Proposal]
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)
			rows := make([][]string, len(resp.Data))
			for i, p := range resp.Data {
				rows[i] = []string{
					p.ID,
					output.Truncate(p.Title, 40),
					output.ColorStatus(p.Status, nc),
					p.OrganizerAgentID,
					fmt.Sprintf("%d", len(p.ParticipantAgentIDs)),
					p.CreatedAt.Format("2006-01-02"),
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "Title", "Status", "Organizer", "Participants", "Created"},
				Rows:    rows,
			}, nc)

			fmt.Printf("\nShowing %d of %d proposals\n", len(resp.Data), resp.Total)
			return nil
		},
	}

	cmd.Flags().String("status", "", "Filter by status (pending, confirmed, expired, cancelled)")
	cmd.Flags().String("organizer", "", "Filter by organizer agent ID")
	cmd.Flags().Int("limit", 50, "Max results to return")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().Bool("all", false, "Fetch all pages automatically")

	return cmd
}

func newSchedulingGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a scheduling proposal by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := fmt.Sprintf("/v1/scheduling/proposals/%s", args[0])
			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			var proposal client.Proposal
			if err := json.Unmarshal(body, &proposal); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, proposal) {
				return nil
			}

			nc := noColor(cmd)
			fmt.Printf("ID:           %s\n", proposal.ID)
			fmt.Printf("Title:        %s\n", proposal.Title)
			fmt.Printf("Status:       %s\n", output.ColorStatus(proposal.Status, nc))
			fmt.Printf("Organizer:    %s\n", proposal.OrganizerAgentID)
			fmt.Printf("Participants: %s\n", strings.Join(proposal.ParticipantAgentIDs, ", "))
			fmt.Printf("Calendar:     %s\n", proposal.CalendarID)
			fmt.Printf("Slots:        %d\n", len(proposal.Slots))
			fmt.Printf("Responses:    %d\n", len(proposal.Responses))
			return nil
		},
	}
}

func newSchedulingRespondCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "respond <id> <@file>",
		Short: "Submit an agent response to a proposal",
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

			path := fmt.Sprintf("/v1/scheduling/proposals/%s/respond", args[0])
			body, _, err := c.Post(path, payload)
			if err != nil {
				return formatError(err)
			}

			var resp client.ProposalResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			fmt.Printf("Response %s recorded: %s from %s\n", resp.ID, resp.Response, resp.AgentID)
			return nil
		},
	}
	return cmd
}

func newSchedulingResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <id>",
		Short: "Resolve a proposal (pick winning slot and create event)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := fmt.Sprintf("/v1/scheduling/proposals/%s/resolve", args[0])
			body, _, err := c.Post(path, nil)
			if err != nil {
				return formatError(err)
			}

			var result client.ResolveProposalResult
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, result) {
				return nil
			}

			fmt.Printf("Status: %s\n", result.Status)
			if result.ResolvedSlot != nil {
				fmt.Printf("Slot:   %s → %s\n", result.ResolvedSlot.StartTime, result.ResolvedSlot.EndTime)
			}
			if result.Reason != nil {
				fmt.Printf("Reason: %s\n", *result.Reason)
			}
			return nil
		},
	}
}

func newSchedulingCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <id>",
		Short: "Cancel a scheduling proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := fmt.Sprintf("/v1/scheduling/proposals/%s/cancel", args[0])
			body, _, err := c.Post(path, nil)
			if err != nil {
				return formatError(err)
			}

			var result client.CancelProposalResult
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, result) {
				return nil
			}

			fmt.Printf("Proposal %s %s.\n", args[0], result.Status)
			return nil
		},
	}
}
