package cmd

import (
	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultBaseURL = "https://api.chronary.ai"

// NewRootCmd creates the root Cobra command with all subcommands.
func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "chronary",
		Short:         "Chronary CLI — calendar-as-a-service for AI agents",
		Long:          "Manage agents, calendars, events, and more from the command line.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringP("output", "o", "table", "Output format: json, yaml, or table")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging (show HTTP requests/responses)")
	rootCmd.PersistentFlags().String("api-key", "", "API key (overrides env and config)")
	rootCmd.PersistentFlags().String("base-url", "", "API base URL (default: https://api.chronary.ai)")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")

	// Bind flags to Viper
	viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("base_url", rootCmd.PersistentFlags().Lookup("base-url"))
	viper.BindEnv("api_key", "CHRONARY_API_KEY")
	viper.BindEnv("base_url", "CHRONARY_BASE_URL")

	// Register subcommands
	rootCmd.AddCommand(newHealthCmd())
	rootCmd.AddCommand(newVersionCmd(version))
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newKeysCmd())
	rootCmd.AddCommand(newAgentsCmd())
	rootCmd.AddCommand(newCalendarsCmd())
	rootCmd.AddCommand(newEventsCmd())
	rootCmd.AddCommand(newAvailabilityCmd())
	rootCmd.AddCommand(newWebhooksCmd())
	rootCmd.AddCommand(newICalCmd())
	rootCmd.AddCommand(newSchedulingCmd())
	rootCmd.AddCommand(newUsageCmd())
	rootCmd.AddCommand(newAuditLogCmd())
	rootCmd.AddCommand(newFeedbackCmd())
	rootCmd.AddCommand(newPlansCmd())
	rootCmd.AddCommand(newTermsCmd())
	rootCmd.AddCommand(newConnectionLinksCmd())

	return rootCmd
}

// clientFromCmd builds an API client from the resolved flags/config/env.
func clientFromCmd(cmd *cobra.Command) (*client.Client, error) {
	cfg, _ := config.Load()

	apiKey := viper.GetString("api_key")
	baseURL := viper.GetString("base_url")

	// Resolve from profile if not set via flag/env
	if cfg != nil {
		profile := cfg.ActiveOrDefault()
		if apiKey == "" {
			apiKey = profile.APIKey
		}
		if baseURL == "" && profile.BaseURL != "" {
			baseURL = profile.BaseURL
		}
	}

	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	debug, _ := cmd.Flags().GetBool("debug")

	return client.NewClient(baseURL, apiKey, debug), nil
}

// outputFormat returns the resolved output format flag.
func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Flags().GetString("output")
	if f != "json" && f != "yaml" && f != "table" {
		return "table"
	}
	return f
}

// noColor returns whether colored output is disabled.
func noColor(cmd *cobra.Command) bool {
	nc, _ := cmd.Flags().GetBool("no-color")
	return nc
}
