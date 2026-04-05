package core

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

// AppConfig represents the configuration from config/app.json
type AppConfig struct {
	Name           string         `json:"name"`
	Package        string         `json:"package"`
	Version        string         `json:"version"`
	VersionCode    int            `json:"version_code"`
	ServerPort     string         `json:"server_port"`
	DefaultDNS     string         `json:"default_dns"`
	DefaultMTU     int            `json:"default_mtu"`
	PingURL        string         `json:"ping_url"`
	GitHubRepo     string         `json:"github_repo"`
	InstallDir     string         `json:"install_dir"`
	Branding       BrandingConfig `json:"branding"`
	Icon           IconConfig     `json:"icon"`
	SupportedLangs []string       `json:"supported_languages"`
	DefaultLang    string         `json:"default_language"`
}

// BrandingConfig represents branding settings
type BrandingConfig struct {
	PrimaryColor string `json:"primary_color"`
	PrimaryDark  string `json:"primary_dark"`
	AccentColor  string `json:"accent_color"`
	NotifChannel string `json:"notification_channel"`
}

// IconConfig represents icon settings
type IconConfig struct {
	BackgroundColor string `json:"background_color"`
	ForegroundColor string `json:"foreground_color"`
}

// RuntimeCfg represents the runtime configuration (config.json)
type RuntimeCfg struct {
	Role         string `json:"role"`
	Password     string `json:"password"`
	ServerListen string `json:"server_listen,omitempty"`
	ClientPeer   string `json:"client_peer,omitempty"`
	ServerPort   string `json:"server_port,omitempty"`
	DefaultDNS   string `json:"default_dns,omitempty"`
	DefaultMTU   int    `json:"default_mtu,omitempty"`
	PingURL      string `json:"ping_url,omitempty"`
}

// LoadAppConfig loads configuration from app.json
func LoadAppConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read app config: %w", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse app config: %w", err)
	}

	// Apply defaults for empty values
	if cfg.ServerPort == "" {
		cfg.ServerPort = DefPort
	}
	if cfg.DefaultMTU == 0 {
		cfg.DefaultMTU = 1500
	}
	if cfg.PingURL == "" {
		cfg.PingURL = "https://cp.cloudflare.com"
	}
	if cfg.DefaultDNS == "" {
		cfg.DefaultDNS = "1.1.1.1"
	}

	return &cfg, nil
}

// LoadRuntimeCfg loads runtime configuration from config.json
func LoadRuntimeCfg(path string) (*RuntimeCfg, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read runtime config: %w", err)
	}

	var cfg RuntimeCfg
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse runtime config: %w", err)
	}

	return &cfg, nil
}

// CliOverrides holds typed CLI flag overrides.
// Nil/zero values mean "not provided".
type CliOverrides struct {
	ServerPort string
	DefaultDNS string
	DefaultMTU int
	PingURL    string
}

// MergeConfig merges app config, runtime config, and CLI overrides.
// Priority: CLI flags > runtime config > app config defaults
func MergeConfig(appCfg *AppConfig, runtimeCfg *RuntimeCfg, cli *CliOverrides) *RuntimeCfg {
	result := &RuntimeCfg{}

	if runtimeCfg != nil {
		*result = *runtimeCfg
	}

	if appCfg != nil {
		if result.ServerPort == "" {
			result.ServerPort = appCfg.ServerPort
		}
		if result.DefaultDNS == "" {
			result.DefaultDNS = appCfg.DefaultDNS
		}
		if result.DefaultMTU == 0 {
			result.DefaultMTU = appCfg.DefaultMTU
		}
		if result.PingURL == "" {
			result.PingURL = appCfg.PingURL
		}
	}

	if cli != nil {
		if cli.ServerPort != "" {
			result.ServerPort = cli.ServerPort
		}
		if cli.DefaultDNS != "" {
			result.DefaultDNS = cli.DefaultDNS
		}
		if cli.DefaultMTU > 0 {
			result.DefaultMTU = cli.DefaultMTU
		}
		if cli.PingURL != "" {
			result.PingURL = cli.PingURL
		}
	}

	return result
}

// Validate checks the runtime configuration for common errors.
func (c *RuntimeCfg) Validate() error {
	if c.ServerPort != "" {
		port, err := strconv.Atoi(c.ServerPort)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid server port: %s", c.ServerPort)
		}
	}
	if c.DefaultDNS != "" {
		if ip := net.ParseIP(c.DefaultDNS); ip == nil {
			return fmt.Errorf("invalid DNS address: %s", c.DefaultDNS)
		}
	}
	if c.PingURL != "" {
		if _, err := url.ParseRequestURI(c.PingURL); err != nil {
			return fmt.Errorf("invalid ping URL: %s", c.PingURL)
		}
	}
	if c.DefaultMTU < 576 || c.DefaultMTU > 9000 {
		return fmt.Errorf("invalid MTU: %d (must be 576-9000)", c.DefaultMTU)
	}
	return nil
}

// FindAppConfigPath attempts to find app.json in common locations
func FindAppConfigPath() string {
	locations := []string{
		"config/app.json",
		"../config/app.json",
		"../../config/app.json",
	}

	for _, loc := range locations {
		if abs, err := filepath.Abs(loc); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	return ""
}

// SaveRuntimeCfg saves the runtime configuration to config.json
func SaveRuntimeCfg(path string, cfg *RuntimeCfg) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal runtime config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write runtime config: %w", err)
	}

	return os.Rename(tmp, path)
}

// ParseInt parses an integer string. Returns 0 on error.
func ParseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
