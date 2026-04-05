package golib

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/lionheart-vpn/lionheart/core"
)

// Tools provides utility functions for Android
type Tools struct{}

// NewTools creates a new Tools instance
func NewTools() *Tools {
	return &Tools{}
}

// ParseSmartKey parses a smart key and returns server info
func (t *Tools) ParseSmartKey(smartKey string) (map[string]string, error) {
	peer, password, err := core.ParseSmartKey(smartKey)
	if err != nil {
		return nil, err
	}

	host, port, err := net.SplitHostPort(peer)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"peer":     peer,
		"host":     host,
		"port":     port,
		"password": password,
	}, nil
}

// GenerateSmartKey generates a smart key from server info
func (t *Tools) GenerateSmartKey(serverIP, port, password string) string {
	return core.EncodeSmartKey(serverIP, port, password)
}

// ValidateSmartKey validates a smart key format
func (t *Tools) ValidateSmartKey(smartKey string) bool {
	_, _, err := core.ParseSmartKey(smartKey)
	return err == nil
}

// GetServerIPFromSmartKey extracts server IP from smart key
func (t *Tools) GetServerIPFromSmartKey(smartKey string) string {
	ip, err := core.SmartKeyServerIP(smartKey)
	if err != nil {
		return ""
	}
	return ip
}

// GetRoutingPresets returns available routing presets
func (t *Tools) GetRoutingPresets() string {
	presets := core.GetPresetWithDescription()
	data, _ := json.Marshal(presets)
	return string(data)
}

// GetRoutingPreset returns a specific routing preset
func (t *Tools) GetRoutingPreset(name string) string {
	preset := core.GetPreset(name)
	if preset == nil {
		return ""
	}
	data, _ := json.Marshal(preset)
	return string(data)
}

// CreateCustomRoutingRules creates custom routing rules
func (t *Tools) CreateCustomRoutingRules(
	geoIPDirect, geoIPProxy, geoIPBlock []string,
	geoSiteDirect, geoSiteProxy, geoSiteBlock []string,
	final string,
) string {
	rules := &core.RoutingRules{
		GeoIPDirect:   geoIPDirect,
		GeoIPProxy:    geoIPProxy,
		GeoIPBlock:    geoIPBlock,
		GeoSiteDirect: geoSiteDirect,
		GeoSiteProxy:  geoSiteProxy,
		GeoSiteBlock:  geoSiteBlock,
		Final:         final,
	}
	data, _ := json.Marshal(rules)
	return string(data)
}

// MergeRoutingRules merges multiple routing rules
func (t *Tools) MergeRoutingRules(rulesJSON ...string) string {
	rules := make([]*core.RoutingRules, 0, len(rulesJSON))
	for _, r := range rulesJSON {
		var rule core.RoutingRules
		if err := json.Unmarshal([]byte(r), &rule); err == nil {
			rules = append(rules, &rule)
		}
	}
	merged := core.MergeRules(rules...)
	data, _ := json.Marshal(merged)
	return string(data)
}

// ExportSingBoxConfig exports a sing-box configuration
func (t *Tools) ExportSingBoxConfig(
	serverIP string,
	port int,
	password string,
	routingRulesJSON string,
) string {
	var rules *core.RoutingRules
	if routingRulesJSON != "" {
		json.Unmarshal([]byte(routingRulesJSON), &rules)
	}

	smartKey := core.EncodeSmartKey(serverIP, fmt.Sprintf("%d", port), password)
	config := core.CreateDefaultConfig(smartKey, serverIP, port, password, rules)

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

// GetVersion returns the core version
func (t *Tools) GetVersion() string {
	return core.Version
}

// GetSingBoxVersion returns the sing-box version
func (t *Tools) GetSingBoxVersion() string {
	return core.SingBoxVersion
}

// FormatBytes formats bytes to human-readable string
func (t *Tools) FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatDuration formats duration to human-readable string
func (t *Tools) FormatDuration(seconds int) string {
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if d > 0 {
		return fmt.Sprintf("%dd %02dh %02dm %02ds", d, h, m, s)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// IsPrivateIP checks if an IP is private
func (t *Tools) IsPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// GetLocalIP returns the local IP address
func (t *Tools) GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			ip := ipNet.IP
			if !ip.IsLoopback() && ip.To4() != nil && !t.IsPrivateIP(ip.String()) {
				return ip.String()
			}
		}
	}

	// Fallback to private IP
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			ip := ipNet.IP
			if !ip.IsLoopback() && ip.To4() != nil {
				return ip.String()
			}
		}
	}

	return "127.0.0.1"
}

// Base64Encode encodes string to base64
func (t *Tools) Base64Encode(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

// Base64Decode decodes base64 string
func (t *Tools) Base64Decode(s string) string {
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(data)
}

// TruncateString truncates string to max length
func (t *Tools) TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// MaskIP masks IP address (e.g., 1.2.3.4 -> 1.2.x.x)
func (t *Tools) MaskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + "." + parts[1] + ".x.x"
}

// ValidateIP validates IP address
func (t *Tools) ValidateIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ValidatePort validates port number
func (t *Tools) ValidatePort(port int) bool {
	return port > 0 && port <= 65535
}

// GetDefaultDNS returns default DNS servers
func (t *Tools) GetDefaultDNS() string {
	servers := []string{
		"1.1.1.1",        // Cloudflare
		"8.8.8.8",        // Google
		"94.140.14.14",   // AdGuard
		"208.67.222.222", // OpenDNS
	}
	data, _ := json.Marshal(servers)
	return string(data)
}

// GetAdGuardDNS returns AdGuard DNS servers
func (t *Tools) GetAdGuardDNS() string {
	return `{
		"standard": "94.140.14.14",
		"family": "94.140.14.15",
		"non_filtering": "94.140.14.140"
	}`
}
