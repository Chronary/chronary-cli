package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newConnectionLinksCmd() *cobra.Command {
	root := &cobra.Command{Use: "connection-links", Short: "Request and inspect human calendar setup links"}
	root.AddCommand(newConnectionLinkCreateCmd(), newConnectionLinkGetCmd(), newConnectionLinkCancelCmd())
	return root
}

func newConnectionLinkCreateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "create", Short: "Create a one-time human calendar setup link", RunE: func(cmd *cobra.Command, _ []string) error {
		c, err := clientFromCmd(cmd)
		if err != nil {
			return err
		}
		if err = requireAPIKey(c); err != nil {
			return err
		}
		calendarID, _ := cmd.Flags().GetString("calendar")
		capabilities, _ := cmd.Flags().GetString("capabilities")
		policy, _ := cmd.Flags().GetString("publication-policy")
		if calendarID == "" {
			return fmt.Errorf("--calendar is required")
		}
		payload := map[string]any{"capabilities": strings.Split(capabilities, ","), "publication_policy": policy}
		path := fmt.Sprintf("/v1/calendars/%s/connection-links", calendarID)
		body, _, err := c.Post(path, payload)
		if err != nil {
			return formatError(err)
		}
		var result map[string]any
		if err = json.Unmarshal(body, &result); err != nil {
			return err
		}
		if printStructured(cmd, result) {
			return nil
		}
		fmt.Printf("Setup link: %v\n", result["setup_url"])
		return nil
	}}
	cmd.Flags().String("calendar", "", "Chronary calendar ID")
	cmd.Flags().String("capabilities", "availability", "Comma-separated: availability,publishing")
	cmd.Flags().String("publication-policy", "none", "none, confirmed, or confirmed_tentative")
	return cmd
}

func newConnectionLinkGetCmd() *cobra.Command {
	return &cobra.Command{Use: "get <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromCmd(cmd)
		if err != nil {
			return err
		}
		body, _, err := c.Get("/v1/connection-links/" + args[0])
		if err != nil {
			return formatError(err)
		}
		var result map[string]any
		if err = json.Unmarshal(body, &result); err != nil {
			return err
		}
		printStructured(cmd, result)
		return nil
	}}
}

func newConnectionLinkCancelCmd() *cobra.Command {
	return &cobra.Command{Use: "cancel <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromCmd(cmd)
		if err != nil {
			return err
		}
		_, _, err = c.Delete("/v1/connection-links/" + args[0])
		return formatError(err)
	}}
}
