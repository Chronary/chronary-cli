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

func newAgentsCmd() *cobra.Command {
	agentsCmd := &cobra.Command{
		Use:     "agents",
		Aliases: []string{"agent"},
		Short:   "Manage agents",
	}

	agentsCmd.AddCommand(newAgentsListCmd())
	agentsCmd.AddCommand(newAgentsCreateCmd())
	agentsCmd.AddCommand(newAgentsGetCmd())
	agentsCmd.AddCommand(newAgentsUpdateCmd())
	agentsCmd.AddCommand(newAgentsDeleteCmd())

	return agentsCmd
}

func newAgentsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			// Build query params (excluding limit/offset when --all is used)
			all, _ := cmd.Flags().GetBool("all")
			params := []string{}
			if v, _ := cmd.Flags().GetString("type"); v != "" {
				params = append(params, "type="+v)
			}
			if v, _ := cmd.Flags().GetString("status"); v != "" {
				params = append(params, "status="+v)
			}

			path := "/v1/agents"

			var body []byte
			if all {
				if len(params) > 0 {
					path += "?" + strings.Join(params, "&")
				}
				items, total, err := fetchAllPages(c, path, 200)
				if err != nil {
					return formatError(err)
				}
				body, err = rewrapList(items, total)
				if err != nil {
					return fmt.Errorf("building response: %w", err)
				}
			} else {
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

			var resp client.ListResponse[client.Agent]
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)

			rows := make([][]string, len(resp.Data))
			for i, a := range resp.Data {
				desc := ""
				if a.Description != nil {
					desc = output.Truncate(*a.Description, 40)
				}
				rows[i] = []string{
					a.ID,
					a.Name,
					a.Type,
					output.ColorStatus(a.Status, nc),
					desc,
					a.CreatedAt.Format("2006-01-02"),
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "Name", "Type", "Status", "Description", "Created"},
				Rows:    rows,
			}, nc)

			fmt.Printf("\nShowing %d of %d agents\n", len(resp.Data), resp.Total)
			return nil
		},
	}

	cmd.Flags().String("type", "", "Filter by type (ai, human, resource)")
	cmd.Flags().String("status", "", "Filter by status (active, paused, decommissioned)")
	cmd.Flags().Int("limit", 50, "Max results to return")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().Bool("all", false, "Fetch all pages automatically")

	return cmd
}

func newAgentsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [@file]",
		Short: "Create a new agent",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			// Check for @file input
			payload, _ := checkFileArg(args, 0)
			if payload == nil {
				name, _ := cmd.Flags().GetString("name")
				agentType, _ := cmd.Flags().GetString("type")
				description, _ := cmd.Flags().GetString("description")
				metadataStr, _ := cmd.Flags().GetString("metadata")

				payload = map[string]any{
					"name": name,
					"type": agentType,
				}
				if description != "" {
					payload["description"] = description
				}
				if metadataStr != "" {
					var meta map[string]any
					if err := json.Unmarshal([]byte(metadataStr), &meta); err != nil {
						return fmt.Errorf("--metadata must be valid JSON: %w", err)
					}
					payload["metadata"] = meta
				}
			}

			body, _, err := c.Post("/v1/agents", payload)
			if err != nil {
				return formatError(err)
			}

			var agent client.Agent
			if err := json.Unmarshal(body, &agent); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, agent) {
				return nil
			}

			nc := noColor(cmd)
			fmt.Printf("Created agent %s (%s)\n", agent.ID, output.ColorStatus(agent.Status, nc))
			return nil
		},
	}

	cmd.Flags().String("name", "", "Agent name (required)")
	cmd.Flags().String("type", "ai", "Agent type: ai, human, or resource")
	cmd.Flags().String("description", "", "Agent description")
	cmd.Flags().String("metadata", "", "Agent metadata as JSON string")
	cmd.MarkFlagRequired("name")

	return cmd
}

func newAgentsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an agent by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Get("/v1/agents/" + args[0])
			if err != nil {
				return formatError(err)
			}

			var agent client.Agent
			if err := json.Unmarshal(body, &agent); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, agent) {
				return nil
			}

			nc := noColor(cmd)
			desc := ""
			if agent.Description != nil {
				desc = *agent.Description
			}

			fmt.Printf("ID:          %s\n", agent.ID)
			fmt.Printf("Name:        %s\n", agent.Name)
			fmt.Printf("Type:        %s\n", agent.Type)
			fmt.Printf("Status:      %s\n", output.ColorStatus(agent.Status, nc))
			if desc != "" {
				fmt.Printf("Description: %s\n", desc)
			}
			fmt.Printf("Created:     %s\n", agent.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated:     %s\n", agent.UpdatedAt.Format("2006-01-02 15:04:05"))
			if len(agent.Metadata) > 0 && string(agent.Metadata) != "{}" && string(agent.Metadata) != "null" {
				fmt.Printf("Metadata:    %s\n", string(agent.Metadata))
			}
			return nil
		},
	}
}

func newAgentsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an agent",
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
			if cmd.Flags().Changed("status") {
				v, _ := cmd.Flags().GetString("status")
				payload["status"] = v
			}
			if cmd.Flags().Changed("description") {
				v, _ := cmd.Flags().GetString("description")
				payload["description"] = v
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
				return fmt.Errorf("at least one flag required: --name, --status, --description, --metadata")
			}

			body, _, err := c.Patch("/v1/agents/"+args[0], payload)
			if err != nil {
				return formatError(err)
			}

			var agent client.Agent
			if err := json.Unmarshal(body, &agent); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, agent) {
				return nil
			}

			nc := noColor(cmd)
			fmt.Printf("Updated agent %s (%s)\n", agent.ID, output.ColorStatus(agent.Status, nc))
			return nil
		},
	}

	cmd.Flags().String("name", "", "New agent name")
	cmd.Flags().String("status", "", "New status: active or paused")
	cmd.Flags().String("description", "", "New description")
	cmd.Flags().String("metadata", "", "New metadata as JSON string")

	return cmd
}

func newAgentsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete (decommission) an agent",
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
					Title(fmt.Sprintf("Delete agent %s?", args[0])).
					Description("This will decommission the agent. This action cannot be undone.").
					Value(&confirm).
					Run()
				if err != nil || !confirm {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			_, _, err = c.Delete("/v1/agents/" + args[0])
			if err != nil {
				return formatError(err)
			}

			fmt.Printf("Agent %s deleted.\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (alias for --force)")

	return cmd
}
