package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newConnectionLinksCmd() *cobra.Command {
	root := &cobra.Command{Use: "connection-links", Short: "Request and inspect human calendar setup links"}
	root.AddCommand(newConnectionLinkCreateCmd(), newConnectionLinkListCmd(), newConnectionLinkGetCmd(), newConnectionLinkCancelCmd())
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

func newConnectionLinkListCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "list", Short: "List human calendar setup links, newest first", RunE: func(cmd *cobra.Command, _ []string) error {
		c, err := clientFromCmd(cmd)
		if err != nil {
			return err
		}
		params := url.Values{}
		if v, _ := cmd.Flags().GetString("status"); v != "" {
			params.Set("status", v)
		}
		if v, _ := cmd.Flags().GetString("calendar"); v != "" {
			params.Set("calendar_id", v)
		}
		if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
			params.Set("limit", strconv.Itoa(v))
		}
		path := "/v1/connection-links"
		path = appendQueryParams(path, params)
		body, _, err := c.Get(path)
		if err != nil {
			return formatError(err)
		}
		var result struct {
			Data []map[string]any `json:"data"`
		}
		if err = json.Unmarshal(body, &result); err != nil {
			return err
		}
		if printStructured(cmd, result) {
			return nil
		}
		if len(result.Data) == 0 {
			fmt.Println("No connection links.")
			return nil
		}
		for _, link := range result.Data {
			fmt.Printf("%v\t%v\t%v\t%v\n", link["id"], link["calendar_id"], link["status"], link["expires_at"])
		}
		return nil
	}}
	cmd.Flags().String("status", "", "awaiting_human, in_progress, completed, declined, expired, or cancelled")
	cmd.Flags().String("calendar", "", "Only links for this Chronary calendar ID")
	cmd.Flags().Int("limit", 0, "Maximum links to return (1-100, default 50)")
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
