package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"claudecode/internal/core"
)

type Config struct {
	APIKey      string
	BaseURL     string
	Model       string
	ProjectDir  string
	HomeDir     string
	Permissions PermissionConfig
}

type PermissionConfig struct {
	Mode         string
	AllowedTools []string
	DeniedTools  []string
}

type fileSettings struct {
	APIKey      *string             `json:"api_key,omitempty"`
	BaseURL     *string             `json:"base_url,omitempty"`
	Model       *string             `json:"model,omitempty"`
	Permissions *filePermissions    `json:"permissions,omitempty"`
}

type filePermissions struct {
	Mode  *string  `json:"mode,omitempty"`
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

func Load() (*Config, error) {
	cfg := &Config{
		BaseURL: "https://api.anthropic.com",
		Model:   "claude-opus-4-7",
		Permissions: PermissionConfig{
			Mode: "allow",
		},
	}

	home, err := os.UserHomeDir()
	if err == nil {
		cfg.HomeDir = home
	}

	if cfg.HomeDir != "" {
		userPath := filepath.Join(cfg.HomeDir, ".claude", "settings.json")
		if s, ok := readSettings(userPath); ok {
			applySettings(cfg, s)
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		cfg.ProjectDir = cwd
		projectPath := filepath.Join(cwd, ".claude", "settings.json")
		if s, ok := readSettings(projectPath); ok {
			applySettings(cfg, s)
		}
	}

	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("ANTHROPIC_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("CLAUDECODE_MODEL"); v != "" {
		cfg.Model = v
	}

	if cfg.APIKey == "" {
		return nil, core.ErrConfigMissing
	}

	if cfg.Permissions.Mode == "" {
		cfg.Permissions.Mode = "allow"
	}

	return cfg, nil
}

func readSettings(path string) (*fileSettings, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "config: read %s: %v\n", path, err)
		}
		return nil, false
	}
	var s fileSettings
	if err := json.Unmarshal(data, &s); err != nil {
		fmt.Fprintf(os.Stderr, "config: parse %s: %v\n", path, err)
		return nil, false
	}
	return &s, true
}

func applySettings(cfg *Config, s *fileSettings) {
	if s.APIKey != nil {
		cfg.APIKey = *s.APIKey
	}
	if s.BaseURL != nil {
		cfg.BaseURL = *s.BaseURL
	}
	if s.Model != nil {
		cfg.Model = *s.Model
	}
	if s.Permissions != nil {
		if s.Permissions.Mode != nil {
			cfg.Permissions.Mode = *s.Permissions.Mode
		}
		if s.Permissions.Allow != nil {
			cfg.Permissions.AllowedTools = s.Permissions.Allow
		}
		if s.Permissions.Deny != nil {
			cfg.Permissions.DeniedTools = s.Permissions.Deny
		}
	}
}
