package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Profile represents a single named authentication profile.
type Profile struct {
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

// Config represents the persisted CLI configuration.
type Config struct {
	// Legacy single-profile fields (kept for backward compatibility on load)
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`

	ActiveProfile string              `json:"active_profile,omitempty"`
	Profiles      map[string]*Profile `json:"profiles,omitempty"`
}

// ActiveOrDefault returns the active profile, falling back to legacy fields.
func (c *Config) ActiveOrDefault() *Profile {
	if c.Profiles != nil && c.ActiveProfile != "" {
		if p, ok := c.Profiles[c.ActiveProfile]; ok {
			return p
		}
	}
	// Fallback to legacy top-level fields
	return &Profile{APIKey: c.APIKey, BaseURL: c.BaseURL}
}

// SetProfile stores a profile by name and sets it as active.
func (c *Config) SetProfile(name string, p *Profile) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}
	c.Profiles[name] = p
	c.ActiveProfile = name
}

// RemoveProfile deletes a profile by name. Returns false if not found.
func (c *Config) RemoveProfile(name string) bool {
	if c.Profiles == nil {
		return false
	}
	if _, ok := c.Profiles[name]; !ok {
		return false
	}
	delete(c.Profiles, name)
	if c.ActiveProfile == name {
		c.ActiveProfile = ""
	}
	return true
}

// Migrate moves legacy top-level fields into a "default" profile if profiles don't exist yet.
func (c *Config) Migrate() {
	if c.Profiles != nil || c.APIKey == "" {
		return
	}
	c.Profiles = map[string]*Profile{
		"default": {APIKey: c.APIKey, BaseURL: c.BaseURL},
	}
	c.ActiveProfile = "default"
	c.APIKey = ""
	c.BaseURL = ""
}

// Dir returns the configuration directory path.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "chronary"), nil
}

// Path returns the full path to the config file.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk. Returns nil config (no error) if file doesn't exist.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadFrom(p)
}

// LoadFrom reads config from a specific path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the config to disk, creating the directory if needed.
func Save(cfg *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	return SaveTo(p, cfg)
}

// SaveTo writes config to a specific path.
func SaveTo(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
