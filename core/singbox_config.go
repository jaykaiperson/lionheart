// Package core provides sing-box integration for Lionheart VPN
// with routing rules support
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/settings"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/debug"
)

const (
	SingBoxVersion = "1.11.0"
	ConfigVersion  = "1.0"
)

// RoutingRules defines traffic routing configuration
type RoutingRules struct {
	// GeoIP rules
	GeoIPDirect []string `json:"geoip_direct,omitempty"` // Countries to route directly
	GeoIPProxy  []string `json:"geoip_proxy,omitempty"`  // Countries to route through proxy
	GeoIPBlock  []string `json:"geoip_block,omitempty"`  // Countries to block

	// GeoSite rules
	GeoSiteDirect []string `json:"geosite_direct,omitempty"` // Categories to route directly
	GeoSiteProxy  []string `json:"geosite_proxy,omitempty"`  // Categories to route through proxy
	GeoSiteBlock  []string `json:"geosite_block,omitempty"`  // Categories to block

	// Domain rules
	DomainDirect []string `json:"domain_direct,omitempty"` // Domains to route directly
	DomainProxy  []string `json:"domain_proxy,omitempty"`  // Domains to route through proxy
	DomainBlock  []string `json:"domain_block,omitempty"`  // Domains to block

	// IP rules
	IPDirect []string `json:"ip_direct,omitempty"` // IPs/CIDRs to route directly
	IPProxy  []string `json:"ip_proxy,omitempty"`  // IPs/CIDRs to route through proxy
	IPBlock  []string `json:"ip_block,omitempty"`  // IPs/CIDRs to block

	// Port rules
	PortDirect []string `json:"port_direct,omitempty"` // Ports to route directly
	PortProxy  []string `json:"port_proxy,omitempty"`  // Ports to route through proxy
	PortBlock  []string `json:"port_block,omitempty"`  // Ports to block

	// Protocol rules
	ProtocolDirect []string `json:"protocol_direct,omitempty"` // Protocols to route directly
	ProtocolProxy  []string `json:"protocol_proxy,omitempty"`  // Protocols to route through proxy
	ProtocolBlock  []string `json:"protocol_block,omitempty"`  // Protocols to block

	// Final action (direct, proxy, block)
	Final string `json:"final,omitempty"`
}

// SingBoxConfig represents the complete sing-box configuration
type SingBoxConfig struct {
	Log       *LogConfig       `json:"log,omitempty"`
	DNS       *DNSConfig       `json:"dns,omitempty"`
	NTP       *NTPConfig       `json:"ntp,omitempty"`
	Inbounds  []InboundConfig  `json:"inbounds,omitempty"`
	Outbounds []OutboundConfig `json:"outbounds,omitempty"`
	Route     *RouteConfig     `json:"route,omitempty"`
	Experimental *ExperimentalConfig `json:"experimental,omitempty"`
}

// LogConfig configures logging
type LogConfig struct {
	Disabled  bool   `json:"disabled,omitempty"`
	Level     string `json:"level,omitempty"`      // trace, debug, info, warn, error, fatal, panic
	Output    string `json:"output,omitempty"`     // file path or empty for stdout
	Timestamp bool   `json:"timestamp,omitempty"`  // enable timestamps
}

// DNSConfig configures DNS servers and rules
type DNSConfig struct {
	Servers         []DNSServerConfig `json:"servers,omitempty"`
	Rules           []DNSRuleConfig   `json:"rules,omitempty"`
	Final           string            `json:"final,omitempty"`
	Strategy        string            `json:"strategy,omitempty"`         // ipv4_only, ipv6_only, prefer_ipv4, prefer_ipv6
	DisableCache    bool              `json:"disable_cache,omitempty"`
	DisableExpire   bool              `json:"disable_expire,omitempty"`
	IndependentCache bool             `json:"independent_cache,omitempty"`
	ReverseMapping  bool              `json:"reverse_mapping,omitempty"`
	FakeIP          *FakeIPConfig     `json:"fakeip,omitempty"`
}

// DNSServerConfig defines a DNS server
type DNSServerConfig struct {
	Tag             string `json:"tag"`
	Address         string `json:"address"`
	AddressResolver string `json:"address_resolver,omitempty"`
	AddressStrategy string `json:"address_strategy,omitempty"`
	Strategy        string `json:"strategy,omitempty"`
	Detour          string `json:"detour,omitempty"`
}

// DNSRuleConfig defines DNS routing rules
type DNSRuleConfig struct {
	Type           string   `json:"type,omitempty"`
	Mode           string   `json:"mode,omitempty"`
	Rules          []string `json:"rules,omitempty"`
	Server         string   `json:"server,omitempty"`
	DisableCache   bool     `json:"disable_cache,omitempty"`
	RewriteTTL     int      `json:"rewrite_ttl,omitempty"`
	ClientSubnet   string   `json:"client_subnet,omitempty"`

	// Match conditions
	QueryType      []string `json:"query_type,omitempty"`
	Network        string   `json:"network,omitempty"`
	Domain         []string `json:"domain,omitempty"`
	DomainSuffix   []string `json:"domain_suffix,omitempty"`
	DomainKeyword  []string `json:"domain_keyword,omitempty"`
	DomainRegex    []string `json:"domain_regex,omitempty"`
	Geosite        []string `json:"geosite,omitempty"`
	SourceGeoIP    []string `json:"source_geoip,omitempty"`
	GeoIP          []string `json:"geoip,omitempty"`
	SourceIPCIDR   []string `json:"source_ip_cidr,omitempty"`
	IPCIDR         []string `json:"ip_cidr,omitempty"`
	SourcePort     []int    `json:"source_port,omitempty"`
	SourcePortRange []string `json:"source_port_range,omitempty"`
	Port           []int    `json:"port,omitempty"`
	PortRange      []string `json:"port_range,omitempty"`
	ProcessName    []string `json:"process_name,omitempty"`
	ProcessPath    []string `json:"process_path,omitempty"`
	PackageName    []string `json:"package_name,omitempty"`
	User           []string `json:"user,omitempty"`
	UserID         []int32  `json:"user_id,omitempty"`
	Outbound       []string `json:"outbound,omitempty"`
	Invert         bool     `json:"invert,omitempty"`
}

// FakeIPConfig configures fake IP addresses
type FakeIPConfig struct {
	Enabled    bool     `json:"enabled,omitempty"`
	Inet4Range string   `json:"inet4_range,omitempty"`
	Inet6Range string   `json:"inet6_range,omitempty"`
}

// NTPConfig configures NTP client
type NTPConfig struct {
	Enabled        bool   `json:"enabled,omitempty"`
	Server         string `json:"server,omitempty"`
	ServerPort     int    `json:"server_port,omitempty"`
	Interval       string `json:"interval,omitempty"`
	WriteToSystem  bool   `json:"write_to_system,omitempty"`
	Detour         string `json:"detour,omitempty"`
}

// InboundConfig defines incoming connections
type InboundConfig struct {
	Type                    string                 `json:"type"`
	Tag                     string                 `json:"tag,omitempty"`
	Listen                  string                 `json:"listen,omitempty"`
	ListenPort              int                    `json:"listen_port,omitempty"`
	TCPFastOpen             bool                   `json:"tcp_fast_open,omitempty"`
	TCPMultiPath            bool                   `json:"tcp_multi_path,omitempty"`
	UDPFragment             bool                   `json:"udp_fragment,omitempty"`
	UDPFragmentDefault      bool                   `json:"udp_fragment_default,omitempty"`
	SniffEnabled            bool                   `json:"sniff,omitempty"`
	SniffOverrideDestination bool                  `json:"sniff_override_destination,omitempty"`
	SniffTimeout            string                 `json:"sniff_timeout,omitempty"`
	DomainStrategy          string                 `json:"domain_strategy,omitempty"`
	UDPTimeout              int                    `json:"udp_timeout,omitempty"`
	ProxyProtocol           bool                   `json:"proxy_protocol,omitempty"`
	ProxyProtocolAcceptNoTLSS bool                 `json:"proxy_protocol_accept_no_tls,omitempty"`
	SetSystemProxy          bool                   `json:"set_system_proxy,omitempty"`
	MTU                     int                    `json:"mtu,omitempty"`
	GSO                     bool                   `json:"gso,omitempty"`
	PackageName             []string               `json:"package_name,omitempty"`
	Platform                *PlatformConfig        `json:"platform,omitempty"`
	Settings                map[string]interface{} `json:"settings,omitempty"`
}

// PlatformConfig defines platform-specific settings
type PlatformConfig struct {
	HTTPProxy *HTTPProxyConfig `json:"http_proxy,omitempty"`
}

// HTTPProxyConfig configures HTTP proxy on the platform
type HTTPProxyConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Server  string `json:"server,omitempty"`
	Port    int    `json:"port,omitempty"`
}

// OutboundConfig defines outgoing connections
type OutboundConfig struct {
	Type                  string                 `json:"type"`
	Tag                   string                 `json:"tag,omitempty"`
	Server                string                 `json:"server,omitempty"`
	ServerPort            int                    `json:"server_port,omitempty"`
	Detour                string                 `json:"detour,omitempty"`
	BindInterface         string                 `json:"bind_interface,omitempty"`
	Inet4BindAddress      string                 `json:"inet4_bind_address,omitempty"`
	Inet6BindAddress      string                 `json:"inet6_bind_address,omitempty"`
	ProtectPath           string                 `json:"protect_path,omitempty"`
	RoutingMark           int                    `json:"routing_mark,omitempty"`
	ReuseAddr             bool                   `json:"reuse_addr,omitempty"`
	ConnectTimeout        string                 `json:"connect_timeout,omitempty"`
	TCPFastOpen           bool                   `json:"tcp_fast_open,omitempty"`
	DomainStrategy        string                 `json:"domain_strategy,omitempty"`
	FallbackDelay         string                 `json:"fallback_delay,omitempty"`
	Settings              map[string]interface{} `json:"settings,omitempty"`
}

// RouteConfig defines routing rules
type RouteConfig struct {
	GeoIP               *GeoIPConfig     `json:"geoip,omitempty"`
	Geosite             *GeositeConfig   `json:"geosite,omitempty"`
	Rules               []RouteRuleConfig `json:"rules,omitempty"`
	Final               string           `json:"final,omitempty"`
	AutoDetectInterface bool             `json:"auto_detect_interface,omitempty"`
	OverrideAndroidVPN  bool             `json:"override_android_vpn,omitempty"`
	DefaultInterface    string           `json:"default_interface,omitempty"`
	DefaultMark         int              `json:"default_mark,omitempty"`
}

// GeoIPConfig configures GeoIP database
type GeoIPConfig struct {
	Path           string `json:"path,omitempty"`
	DownloadURL    string `json:"download_url,omitempty"`
	DownloadDetour string `json:"download_detour,omitempty"`
}

// GeositeConfig configures Geosite database
type GeositeConfig struct {
	Path           string `json:"path,omitempty"`
	DownloadURL    string `json:"download_url,omitempty"`
	DownloadDetour string `json:"download_detour,omitempty"`
}

// RouteRuleConfig defines a routing rule
type RouteRuleConfig struct {
	Type           string   `json:"type,omitempty"`
	Mode           string   `json:"mode,omitempty"`
	Rules          []string `json:"rules,omitempty"`
	Outbound       string   `json:"outbound,omitempty"`
	Invert         bool     `json:"invert,omitempty"`

	// Match conditions
	Protocol       []string `json:"protocol,omitempty"`
	Network        string   `json:"network,omitempty"`
	Domain         []string `json:"domain,omitempty"`
	DomainSuffix   []string `json:"domain_suffix,omitempty"`
	DomainKeyword  []string `json:"domain_keyword,omitempty"`
	DomainRegex    []string `json:"domain_regex,omitempty"`
	SourceGeoIP    []string `json:"source_geoip,omitempty"`
	GeoIP          []string `json:"geoip,omitempty"`
	Geosite        []string `json:"geosite,omitempty"`
	SourceIPCIDR   []string `json:"source_ip_cidr,omitempty"`
	IPCIDR         []string `json:"ip_cidr,omitempty"`
	SourcePort     []int    `json:"source_port,omitempty"`
	SourcePortRange []string `json:"source_port_range,omitempty"`
	Port           []int    `json:"port,omitempty"`
	PortRange      []string `json:"port_range,omitempty"`
	ProcessName    []string `json:"process_name,omitempty"`
	ProcessPath    []string `json:"process_path,omitempty"`
	PackageName    []string `json:"package_name,omitempty"`
	User           []string `json:"user,omitempty"`
	UserID         []int32  `json:"user_id,omitempty"`
	ClashMode      string   `json:"clash_mode,omitempty"`
	WifiSSID       []string `json:"wifi_ssid,omitempty"`
	WifiBSSID      []string `json:"wifi_bssid,omitempty"`
	RuleSet        []string `json:"rule_set,omitempty"`
	RuleSetIPCIDRMatchSource bool `json:"rule_set_ip_cidr_match_source,omitempty"`
	Inbound        []string `json:"inbound,omitempty"`
}

// ExperimentalConfig contains experimental features
type ExperimentalConfig struct {
	CacheFile *CacheFileConfig `json:"cache_file,omitempty"`
	ClashAPI  *ClashAPIConfig  `json:"clash_api,omitempty"`
	V2RayAPI  *V2RayAPIConfig  `json:"v2ray_api,omitempty"`
}

// CacheFileConfig configures the cache file
type CacheFileConfig struct {
	Enabled             bool   `json:"enabled,omitempty"`
	Path                string `json:"path,omitempty"`
	CacheID             string `json:"cache_id,omitempty"`
	StoreFakeIP         bool   `json:"store_fakeip,omitempty"`
	StoreRDRC           bool   `json:"store_rdrc,omitempty"`
	RDRCTimeout         string `json:"rdrc_timeout,omitempty"`
}

// ClashAPIConfig configures Clash API
type ClashAPIConfig struct {
	ExternalController       string   `json:"external_controller,omitempty"`
	ExternalUI               string   `json:"external_ui,omitempty"`
	ExternalUIDownloadURL    string   `json:"external_ui_download_url,omitempty"`
	ExternalUIDownloadDetour string   `json:"external_ui_download_detour,omitempty"`
	Secret                   string   `json:"secret,omitempty"`
	DefaultMode              string   `json:"default_mode,omitempty"`
	ModeList                 []string `json:"mode_list,omitempty"`
	StoreSelected            bool     `json:"store_selected,omitempty"`
	StoreFakeIP              bool     `json:"store_fakeip,omitempty"`
}

// V2RayAPIConfig configures V2Ray API
type V2RayAPIConfig struct {
	Listen string `json:"listen,omitempty"`
	Stats  *V2RayStatsConfig `json:"stats,omitempty"`
}

// V2RayStatsConfig configures V2Ray stats
type V2RayStatsConfig struct {
	Enabled   bool     `json:"enabled,omitempty"`
	Inbounds  []string `json:"inbounds,omitempty"`
	Outbounds []string `json:"outbounds,omitempty"`
	Users     []string `json:"users,omitempty"`
}

// SingBoxEngine wraps sing-box instance
type SingBoxEngine struct {
	ctx    context.Context
	cancel context.CancelFunc
	box    *box.Box
	config *SingBoxConfig
	logger log.Logger
}

// NewSingBoxEngine creates a new sing-box engine
func NewSingBoxEngine() *SingBoxEngine {
	return &SingBoxEngine{
		logger: log.NewLogger(log.Options{
			Level: log.LevelInfo,
		}),
	}
}

// Initialize initializes the sing-box engine with configuration
func (e *SingBoxEngine) Initialize(config *SingBoxConfig) error {
	e.config = config

	// Convert to sing-box options
	options, err := e.toOptions(config)
	if err != nil {
		return fmt.Errorf("convert config to options: %w", err)
	}

	// Create context
	e.ctx, e.cancel = context.WithCancel(context.Background())

	// Create sing-box instance
	instance, err := box.New(box.Options{
		Context: e.ctx,
		Options: *options,
	})
	if err != nil {
		return fmt.Errorf("create sing-box instance: %w", err)
	}

	e.box = instance
	return nil
}

// Start starts the sing-box engine
func (e *SingBoxEngine) Start() error {
	if e.box == nil {
		return fmt.Errorf("engine not initialized")
	}
	return e.box.Start()
}

// Stop stops the sing-box engine
func (e *SingBoxEngine) Stop() error {
	if e.cancel != nil {
		e.cancel()
	}
	if e.box != nil {
		return e.box.Close()
	}
	return nil
}

// Status returns the current status of the engine
func (e *SingBoxEngine) Status() string {
	if e.box == nil {
		return "not_initialized"
	}
	// Check if context is still valid
	select {
	case <-e.ctx.Done():
		return "stopped"
	default:
		return "running"
	}
}

// toOptions converts SingBoxConfig to sing-box options
func (e *SingBoxEngine) toOptions(config *SingBoxConfig) (*option.Options, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	var options option.Options
	if err := json.Unmarshal(data, &options); err != nil {
		return nil, err
	}

	return &options, nil
}

// CreateDefaultConfig creates a default configuration for Lionheart
func CreateDefaultConfig(smartKey, serverIP string, serverPort int, password string, rules *RoutingRules) *SingBoxConfig {
	config := &SingBoxConfig{
		Log: &LogConfig{
			Level:     "info",
			Timestamp: true,
		},
		DNS: &DNSConfig{
			Servers: []DNSServerConfig{
				{
					Tag:     "local",
					Address: "local",
					Detour:  "direct",
				},
				{
					Tag:     "proxy-dns",
					Address: "tcp://1.1.1.1",
					Detour:  "proxy",
				},
				{
					Tag:     "block",
					Address: "rcode://success",
				},
			},
			Rules: []DNSRuleConfig{
				{
					Geosite: []string{"category-ads-all"},
					Server:  "block",
				},
				{
					Geosite: []string{"cn", "private"},
					Server:  "local",
				},
			},
			Final:    "proxy-dns",
			Strategy: "prefer_ipv4",
		},
		Inbounds: []InboundConfig{
			{
				Type:       "tun",
				Tag:        "tun-in",
				MTU:        9000,
				SniffEnabled: true,
				Settings: map[string]interface{}{
					"address": []string{
						"172.19.0.1/30",
						"fdfe:dcba:9876::1/126",
					},
					"auto_route":              true,
					"strict_route":            false,
					"endpoint_independent_nat": true,
					"udp_timeout":             "5m",
				},
			},
			{
				Type:       "socks",
				Tag:        "socks-in",
				Listen:     "127.0.0.1",
				ListenPort: 1080,
				SniffEnabled: true,
			},
		},
		Outbounds: []OutboundConfig{
			{
				Type: "direct",
				Tag:  "direct",
			},
			{
				Type: "block",
				Tag:  "block",
			},
			{
				Type:       "selector",
				Tag:        "proxy",
				Settings: map[string]interface{}{
					"outbounds": []string{"lionheart-out", "direct"},
					"default":   "lionheart-out",
				},
			},
			{
				Type:       "urltest",
				Tag:        "auto",
				Settings: map[string]interface{}{
					"outbounds": []string{"lionheart-out"},
					"url":       "http://cp.cloudflare.com/generate_204",
					"interval":  "1m",
				},
			},
		},
		Route: &RouteConfig{
			GeoIP: &GeoIPConfig{
				DownloadURL:    "https://github.com/SagerNet/sing-geoip/releases/latest/download/geoip.db",
				DownloadDetour: "direct",
			},
			Geosite: &GeositeConfig{
				DownloadURL:    "https://github.com/SagerNet/sing-geosite/releases/latest/download/geosite.db",
				DownloadDetour: "direct",
			},
			AutoDetectInterface: true,
			Final:               "proxy",
		},
		Experimental: &ExperimentalConfig{
			CacheFile: &CacheFileConfig{
				Enabled:     true,
				StoreFakeIP: true,
			},
		},
	}

	// Add Lionheart outbound (custom KCP+TURN transport)
	lionheartOutbound := OutboundConfig{
		Type:       "custom",
		Tag:        "lionheart-out",
		Settings: map[string]interface{}{
			"type":       "lionheart",
			"server":     serverIP,
			"server_port": serverPort,
			"password":   password,
			"smart_key":  smartKey,
		},
	}
	config.Outbounds = append(config.Outbounds, lionheartOutbound)

	// Apply routing rules if provided
	if rules != nil {
		applyRoutingRules(config, rules)
	}

	return config
}

// applyRoutingRules applies routing rules to configuration
func applyRoutingRules(config *SingBoxConfig, rules *RoutingRules) {
	routeRules := []RouteRuleConfig{}

	// Block rules (highest priority)
	if len(rules.GeoIPBlock) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			GeoIP:    rules.GeoIPBlock,
			Outbound: "block",
		})
	}
	if len(rules.GeoSiteBlock) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Geosite:  rules.GeoSiteBlock,
			Outbound: "block",
		})
	}
	if len(rules.DomainBlock) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Domain:   rules.DomainBlock,
			Outbound: "block",
		})
	}
	if len(rules.IPBlock) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			IPCIDR:   rules.IPBlock,
			Outbound: "block",
		})
	}
	if len(rules.PortBlock) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Port:     parsePorts(rules.PortBlock),
			Outbound: "block",
		})
	}
	if len(rules.ProtocolBlock) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Protocol: rules.ProtocolBlock,
			Outbound: "block",
		})
	}

	// Direct rules
	if len(rules.GeoIPDirect) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			GeoIP:    rules.GeoIPDirect,
			Outbound: "direct",
		})
	}
	if len(rules.GeoSiteDirect) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Geosite:  rules.GeoSiteDirect,
			Outbound: "direct",
		})
	}
	if len(rules.DomainDirect) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Domain:   rules.DomainDirect,
			Outbound: "direct",
		})
	}
	if len(rules.IPDirect) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			IPCIDR:   rules.IPDirect,
			Outbound: "direct",
		})
	}
	if len(rules.PortDirect) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Port:     parsePorts(rules.PortDirect),
			Outbound: "direct",
		})
	}
	if len(rules.ProtocolDirect) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Protocol: rules.ProtocolDirect,
			Outbound: "direct",
		})
	}

	// Proxy rules
	if len(rules.GeoIPProxy) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			GeoIP:    rules.GeoIPProxy,
			Outbound: "proxy",
		})
	}
	if len(rules.GeoSiteProxy) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Geosite:  rules.GeoSiteProxy,
			Outbound: "proxy",
		})
	}
	if len(rules.DomainProxy) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Domain:   rules.DomainProxy,
			Outbound: "proxy",
		})
	}
	if len(rules.IPProxy) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			IPCIDR:   rules.IPProxy,
			Outbound: "proxy",
		})
	}
	if len(rules.PortProxy) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Port:     parsePorts(rules.PortProxy),
			Outbound: "proxy",
		})
	}
	if len(rules.ProtocolProxy) > 0 {
		routeRules = append(routeRules, RouteRuleConfig{
			Protocol: rules.ProtocolProxy,
			Outbound: "proxy",
		})
	}

	config.Route.Rules = routeRules

	// Set final action
	if rules.Final != "" {
		config.Route.Final = rules.Final
	}
}

// parsePorts parses port strings to integers
func parsePorts(ports []string) []int {
	result := []int{}
	for _, p := range ports {
		var port int
		if _, err := fmt.Sscanf(p, "%d", &port); err == nil {
			result = append(result, port)
		}
	}
	return result
}

// SaveConfig saves configuration to file
func SaveConfig(config *SingBoxConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*SingBoxConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config SingBoxConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetConfigDir returns the configuration directory
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "lionheart")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "lionheart")
	default:
		return filepath.Join(home, ".config", "lionheart")
	}
}

// EnsureConfigDir ensures the configuration directory exists
func EnsureConfigDir() error {
	dir := GetConfigDir()
	return os.MkdirAll(dir, 0755)
}
