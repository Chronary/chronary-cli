package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/config"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/Chronary/chronary-cli/pkg/redact"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}

	authCmd.AddCommand(newAuthLoginCmd())
	authCmd.AddCommand(newAuthSignupCmd())
	authCmd.AddCommand(newAuthVerifyCmd())
	authCmd.AddCommand(newAuthWaitlistCmd())
	authCmd.AddCommand(newAuthStatusCmd())
	authCmd.AddCommand(newAuthListCmd())
	authCmd.AddCommand(newAuthSwitchCmd())
	authCmd.AddCommand(newAuthRemoveCmd())

	return authCmd
}

// promptAPIKey asks for the API key. A package variable so tests can stub
// the interactive prompt.
var promptAPIKey = func() (string, error) {
	var apiKey string
	err := huh.NewInput().
		Title("Enter your Chronary API key").
		Description("Starts with chr_sk_").
		Placeholder("chr_sk_...").
		Validate(func(s string) error {
			if !strings.HasPrefix(s, "chr_sk_") {
				return fmt.Errorf("key must start with chr_sk_")
			}
			return nil
		}).
		Value(&apiKey).
		Run()
	return apiKey, err
}

// loginDefaultBaseURL is the URL `auth login` validates against without
// --base-url. A variable so tests never reach the real API.
var loginDefaultBaseURL = defaultBaseURL

// loadConfig wraps config.Load and turns a read or parse failure into an
// error that names the file, so no command silently treats a corrupt config
// as empty and then saves over it (#2562).
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		path, _ := config.Path()
		return nil, fmt.Errorf("config file %s could not be read: %w. Fix or move it and try again; it was not changed", path, err)
	}
	return cfg, nil
}

func newAuthLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with your Chronary API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load first: never prompt for a key we could not save, and never
			// overwrite an unparseable config (#2562).
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if cfg == nil {
				cfg = &config.Config{}
			}

			profileName, _ := cmd.Flags().GetString("profile")
			if profileName == "" {
				profileName = "default"
			}

			apiKey, err := promptAPIKey()
			if err != nil {
				return fmt.Errorf("prompt cancelled")
			}

			// The base URL comes only from --base-url / CHRONARY_BASE_URL, else
			// the default. Never from a saved profile: the active one belongs
			// to another environment, and the target one may already hold a
			// wrong URL from the old behaviour (#2527).
			baseURL := viper.GetString("base_url")
			if baseURL == "" {
				baseURL = loginDefaultBaseURL
			}
			debug, _ := cmd.Flags().GetBool("debug")
			c := client.NewClient(baseURL, apiKey, debug)

			// Validate against an authenticated endpoint: /health ignores the
			// Authorization header, so it accepted any chr_sk_ string (#2559).
			fmt.Println("Validating API key...")
			if _, status, err := c.Get("/v1/usage"); err != nil {
				if status == 401 || status == 403 {
					reason := "invalid or revoked key"
					var apiErr *client.APIError
					if errors.As(err, &apiErr) && strings.TrimSpace(apiErr.Message) != "" {
						reason = strings.TrimSuffix(strings.TrimSpace(apiErr.Message), ".")
						if apiErr.RequestID != "" {
							reason += fmt.Sprintf(" (request_id: %s)", apiErr.RequestID)
						}
					}
					return fmt.Errorf("the API at %s rejected this key; nothing was saved: %s. Pass --base-url if the key belongs to another environment", baseURL, reason)
				}
				return fmt.Errorf("key validation against %s failed; nothing was saved:\n%w", baseURL, formatError(err))
			}

			saveURL := ""
			if baseURL != loginDefaultBaseURL {
				saveURL = baseURL
			}

			cfg.SetProfile(profileName, &config.Profile{
				APIKey:  apiKey,
				BaseURL: saveURL,
			})

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Authenticated. Key saved to profile %q.\n", profileName)
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

			if c.APIKey == "" {
				if printStructured(cmd, map[string]any{"authenticated": false}) {
					return nil
				}
				fmt.Println("Not authenticated. Run `chronary auth login` to configure.")
				return nil
			}

			masked := redact.MaskKey(c.APIKey)

			if printStructured(cmd, map[string]any{
				"authenticated": true,
				"key":           masked,
				"base_url":      c.BaseURL,
			}) {
				return nil
			}

			fmt.Printf("Key:      %s\n", masked)
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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
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
				"profiles": maskedProfiles(cfg.Profiles),
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
				rows = append(rows, []string{
					active,
					name,
					redact.MaskKey(p.APIKey),
				})
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"", "Profile", "Key"},
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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
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

// maskedProfiles copies the saved profiles with every API key masked, so
// structured `auth list` output never prints a stored key in full.
func maskedProfiles(profiles map[string]*config.Profile) map[string]*config.Profile {
	out := make(map[string]*config.Profile, len(profiles))
	for name, p := range profiles {
		if p == nil {
			out[name] = nil
			continue
		}
		out[name] = &config.Profile{APIKey: redact.MaskKey(p.APIKey), BaseURL: p.BaseURL}
	}
	return out
}
