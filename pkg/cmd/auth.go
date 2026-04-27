package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/Chronary/chronary-cli/pkg/config"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}

	authCmd.AddCommand(newAuthLoginCmd())
	authCmd.AddCommand(newAuthSignupCmd())
	authCmd.AddCommand(newAuthVerifyCmd())
	authCmd.AddCommand(newAuthStatusCmd())
	authCmd.AddCommand(newAuthListCmd())
	authCmd.AddCommand(newAuthSwitchCmd())
	authCmd.AddCommand(newAuthRemoveCmd())

	return authCmd
}

func newAuthLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with your Chronary API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			var apiKey string

			err := huh.NewInput().
				Title("Enter your Chronary API key").
				Description("Starts with chr_sk_live_ or chr_sk_test_").
				Placeholder("chr_sk_live_...").
				Validate(func(s string) error {
					if !strings.HasPrefix(s, "chr_sk_live_") && !strings.HasPrefix(s, "chr_sk_test_") {
						return fmt.Errorf("key must start with chr_sk_live_ or chr_sk_test_")
					}
					return nil
				}).
				Value(&apiKey).
				Run()

			if err != nil {
				return fmt.Errorf("prompt cancelled")
			}

			// Resolve base URL
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			c.APIKey = apiKey

			// Validate key by hitting /health with auth
			fmt.Println("Validating API key...")
			_, _, err = c.Get("/health")
			if err != nil {
				return fmt.Errorf("key validation failed: %w", err)
			}

			// Save to config using profile system
			cfg, _ := config.Load()
			if cfg == nil {
				cfg = &config.Config{}
			}

			profileName, _ := cmd.Flags().GetString("profile")
			if profileName == "" {
				profileName = "default"
			}

			baseURL := ""
			if c.BaseURL != defaultBaseURL {
				baseURL = c.BaseURL
			}

			cfg.SetProfile(profileName, &config.Profile{
				APIKey:  apiKey,
				BaseURL: baseURL,
			})

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			mode := "live"
			if strings.HasPrefix(apiKey, "chr_sk_test_") {
				mode = "test"
			}

			fmt.Printf("Authenticated (%s mode). Key saved to profile %q.\n", mode, profileName)
			return nil
		},
	}

	cmd.Flags().String("profile", "default", "Profile name to save credentials under")

	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}

			nc := noColor(cmd)

			if c.APIKey == "" {
				if printStructured(cmd, map[string]any{"authenticated": false}) {
					return nil
				}
				fmt.Println("Not authenticated. Run `chronary auth login` to configure.")
				return nil
			}

			// Mask key: show prefix + first 4 chars after prefix
			masked := maskKey(c.APIKey)
			mode := "live"
			if strings.HasPrefix(c.APIKey, "chr_sk_test_") {
				mode = "test"
			}

			if printStructured(cmd, map[string]any{
				"authenticated": true,
				"key":           masked,
				"mode":          mode,
				"base_url":      c.BaseURL,
			}) {
				return nil
			}

			modeStr := output.ColorStatus(mode, nc)
			fmt.Printf("Key:      %s\n", masked)
			fmt.Printf("Mode:     %s\n", modeStr)
			fmt.Printf("Base URL: %s\n", c.BaseURL)
			return nil
		},
	}
}

func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			if cfg == nil {
				fmt.Println("No config file found. Run `chronary auth login` to get started.")
				return nil
			}

			// Migrate legacy config if needed
			if cfg.Profiles == nil && cfg.APIKey != "" {
				cfg.Migrate()
			}

			if len(cfg.Profiles) == 0 {
				fmt.Println("No profiles configured. Run `chronary auth login` to get started.")
				return nil
			}

			if printStructured(cmd, map[string]any{
				"active":   cfg.ActiveProfile,
				"profiles": cfg.Profiles,
			}) {
				return nil
			}

			nc := noColor(cmd)
			rows := make([][]string, 0, len(cfg.Profiles))
			for name, p := range cfg.Profiles {
				active := ""
				if name == cfg.ActiveProfile {
					active = "*"
				}
				mode := "live"
				if strings.HasPrefix(p.APIKey, "chr_sk_test_") {
					mode = "test"
				}
				rows = append(rows, []string{
					active,
					name,
					output.ColorStatus(mode, nc),
					maskKey(p.APIKey),
				})
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"", "Profile", "Mode", "Key"},
				Rows:    rows,
			}, nc)
			return nil
		},
	}
}

func newAuthSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "switch <profile>",
		Short: "Switch to a different profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			if cfg == nil || cfg.Profiles == nil {
				return fmt.Errorf("no profiles configured. Run `chronary auth login --profile %s` first", args[0])
			}

			name := args[0]
			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("profile %q not found. Run `chronary auth list` to see available profiles", name)
			}

			cfg.ActiveProfile = name
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Switched to profile %q.\n", name)
			return nil
		},
	}
}

func newAuthRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <profile>",
		Short: "Remove a saved profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			if cfg == nil || cfg.Profiles == nil {
				return fmt.Errorf("no profiles configured")
			}

			name := args[0]
			if !cfg.RemoveProfile(name) {
				return fmt.Errorf("profile %q not found", name)
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Profile %q removed.\n", name)
			return nil
		},
	}
}

// maskKey shows the prefix and first 4 chars, hiding the rest.
func maskKey(key string) string {
	// chr_sk_live_xxxx... or chr_sk_test_xxxx...
	prefixes := []string{"chr_sk_live_", "chr_sk_test_"}
	for _, p := range prefixes {
		if strings.HasPrefix(key, p) {
			rest := key[len(p):]
			if len(rest) <= 4 {
				return key
			}
			return p + rest[:4] + "..."
		}
	}
	if len(key) > 8 {
		return key[:8] + "..."
	}
	return key
}
