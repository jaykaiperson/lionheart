package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")

	appCfg := AppConfig{
		Name:        "TestApp",
		ServerPort:  "9090",
		DefaultDNS:  "8.8.8.8",
		DefaultMTU:  1400,
		PingURL:     "https://example.com/ping",
		DefaultLang: "en",
	}

	data, _ := json.Marshal(appCfg)
	os.WriteFile(cfgPath, data, 0644)

	cfg, err := LoadAppConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadAppConfig() error = %v", err)
	}

	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort = %v, want %v", cfg.ServerPort, "9090")
	}
	if cfg.DefaultDNS != "8.8.8.8" {
		t.Errorf("DefaultDNS = %v, want %v", cfg.DefaultDNS, "8.8.8.8")
	}
	if cfg.DefaultMTU != 1400 {
		t.Errorf("DefaultMTU = %v, want %v", cfg.DefaultMTU, 1400)
	}
	if cfg.PingURL != "https://example.com/ping" {
		t.Errorf("PingURL = %v, want %v", cfg.PingURL, "https://example.com/ping")
	}
}

func TestLoadAppConfig_ApplyDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")

	jsonData := `{"name": "TestApp"}`
	os.WriteFile(cfgPath, []byte(jsonData), 0644)

	cfg, err := LoadAppConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadAppConfig() error = %v", err)
	}

	if cfg.ServerPort != DefPort {
		t.Errorf("ServerPort default = %v, want %v", cfg.ServerPort, DefPort)
	}
	if cfg.DefaultMTU != 1500 {
		t.Errorf("DefaultMTU default = %v, want %v", cfg.DefaultMTU, 1500)
	}
	if cfg.PingURL != "https://cp.cloudflare.com" {
		t.Errorf("PingURL default = %v, want %v", cfg.PingURL, "https://cp.cloudflare.com")
	}
	if cfg.DefaultDNS != "1.1.1.1" {
		t.Errorf("DefaultDNS default = %v, want %v", cfg.DefaultDNS, "1.1.1.1")
	}
}

func TestLoadAppConfig_FileNotFound(t *testing.T) {
	_, err := LoadAppConfig("/nonexistent/path/app.json")
	if err == nil {
		t.Error("LoadAppConfig() expected error for nonexistent file, got nil")
	}
}

func TestLoadAppConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")
	os.WriteFile(cfgPath, []byte(`{invalid json}`), 0644)

	_, err := LoadAppConfig(cfgPath)
	if err == nil {
		t.Error("LoadAppConfig() expected error for invalid JSON, got nil")
	}
}

func TestLoadRuntimeCfg(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	runtimeCfg := RuntimeCfg{
		Role:         "server",
		Password:     "testpass",
		ServerListen: "0.0.0.0:8443",
		ServerPort:   "9090",
		DefaultDNS:   "8.8.8.8",
		DefaultMTU:   1400,
		PingURL:      "https://example.com/ping",
	}

	data, _ := json.MarshalIndent(runtimeCfg, "", "  ")
	os.WriteFile(cfgPath, data, 0644)

	cfg, err := LoadRuntimeCfg(cfgPath)
	if err != nil {
		t.Fatalf("LoadRuntimeCfg() error = %v", err)
	}

	if cfg.Role != "server" {
		t.Errorf("Role = %v, want %v", cfg.Role, "server")
	}
	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort = %v, want %v", cfg.ServerPort, "9090")
	}
	if cfg.DefaultMTU != 1400 {
		t.Errorf("DefaultMTU = %v, want %v", cfg.DefaultMTU, 1400)
	}
}

func TestLoadRuntimeCfg_FileNotFound(t *testing.T) {
	cfg, err := LoadRuntimeCfg("/nonexistent/config.json")
	if err != nil {
		t.Errorf("LoadRuntimeCfg() unexpected error for nonexistent file: %v", err)
	}
	if cfg != nil {
		t.Error("LoadRuntimeCfg() expected nil for nonexistent file, got non-nil")
	}
}

func TestMergeConfig_CLIOverrides(t *testing.T) {
	appCfg := &AppConfig{
		ServerPort: "8443",
		DefaultDNS: "1.1.1.1",
		DefaultMTU: 1500,
		PingURL:    "https://cp.cloudflare.com",
	}

	cli := &CliOverrides{
		ServerPort: "9090",
		DefaultDNS: "8.8.4.4",
		DefaultMTU: 1400,
		PingURL:    "https://example.com/ping",
	}

	result := MergeConfig(appCfg, nil, cli)

	if result.ServerPort != "9090" {
		t.Errorf("ServerPort after CLI override = %v, want %v", result.ServerPort, "9090")
	}
	if result.DefaultDNS != "8.8.4.4" {
		t.Errorf("DefaultDNS after CLI override = %v, want %v", result.DefaultDNS, "8.8.4.4")
	}
	if result.DefaultMTU != 1400 {
		t.Errorf("DefaultMTU after CLI override = %v, want %v", result.DefaultMTU, 1400)
	}
	if result.PingURL != "https://example.com/ping" {
		t.Errorf("PingURL after CLI override = %v, want %v", result.PingURL, "https://example.com/ping")
	}
}

func TestMergeConfig_RuntimeConfigTakesPrecedenceOverAppConfig(t *testing.T) {
	appCfg := &AppConfig{
		ServerPort: "8443",
		DefaultDNS: "1.1.1.1",
		DefaultMTU: 1500,
		PingURL:    "https://cp.cloudflare.com",
	}

	runtimeCfg := &RuntimeCfg{
		ServerPort: "7070",
		DefaultDNS: "9.9.9.9",
	}

	result := MergeConfig(appCfg, runtimeCfg, nil)

	if result.ServerPort != "7070" {
		t.Errorf("ServerPort = %v, want %v (runtime should override app config)", result.ServerPort, "7070")
	}
	if result.DefaultDNS != "9.9.9.9" {
		t.Errorf("DefaultDNS = %v, want %v (runtime should override app config)", result.DefaultDNS, "9.9.9.9")
	}
	if result.DefaultMTU != 1500 {
		t.Errorf("DefaultMTU = %v, want %v", result.DefaultMTU, 1500)
	}
	if result.PingURL != "https://cp.cloudflare.com" {
		t.Errorf("PingURL = %v, want %v", result.PingURL, "https://cp.cloudflare.com")
	}
}

func TestMergeConfig_FullPrecedence(t *testing.T) {
	appCfg := &AppConfig{
		ServerPort: "8443",
		DefaultDNS: "1.1.1.1",
		DefaultMTU: 1500,
		PingURL:    "https://cp.cloudflare.com",
	}

	runtimeCfg := &RuntimeCfg{
		ServerPort: "7070",
		DefaultDNS: "9.9.9.9",
		DefaultMTU: 1400,
	}

	cli := &CliOverrides{
		ServerPort: "9999",
	}

	result := MergeConfig(appCfg, runtimeCfg, cli)

	if result.ServerPort != "9999" {
		t.Errorf("ServerPort = %v, want %v (CLI should have highest priority)", result.ServerPort, "9999")
	}
	if result.DefaultDNS != "9.9.9.9" {
		t.Errorf("DefaultDNS = %v, want %v", result.DefaultDNS, "9.9.9.9")
	}
	if result.DefaultMTU != 1400 {
		t.Errorf("DefaultMTU = %v, want %v", result.DefaultMTU, 1400)
	}
	if result.PingURL != "https://cp.cloudflare.com" {
		t.Errorf("PingURL = %v, want %v", result.PingURL, "https://cp.cloudflare.com")
	}
}

func TestMergeConfig_EmptyCLIOverrides(t *testing.T) {
	appCfg := &AppConfig{
		ServerPort: "8443",
		DefaultDNS: "1.1.1.1",
		DefaultMTU: 1500,
		PingURL:    "https://cp.cloudflare.com",
	}

	result := MergeConfig(appCfg, nil, &CliOverrides{})

	if result.ServerPort != "8443" {
		t.Errorf("ServerPort = %v, want %v", result.ServerPort, "8443")
	}
	if result.DefaultDNS != "1.1.1.1" {
		t.Errorf("DefaultDNS = %v, want %v", result.DefaultDNS, "1.1.1.1")
	}
	if result.DefaultMTU != 1500 {
		t.Errorf("DefaultMTU = %v, want %v", result.DefaultMTU, 1500)
	}
	if result.PingURL != "https://cp.cloudflare.com" {
		t.Errorf("PingURL = %v, want %v", result.PingURL, "https://cp.cloudflare.com")
	}
}

func TestMergeConfig_PartialCLIOverrides(t *testing.T) {
	appCfg := &AppConfig{
		ServerPort: "8443",
		DefaultDNS: "1.1.1.1",
		DefaultMTU: 1500,
		PingURL:    "https://cp.cloudflare.com",
	}

	cli := &CliOverrides{
		ServerPort: "9090",
	}

	result := MergeConfig(appCfg, nil, cli)

	if result.ServerPort != "9090" {
		t.Errorf("ServerPort = %v, want %v", result.ServerPort, "9090")
	}
	if result.DefaultDNS != "1.1.1.1" {
		t.Errorf("DefaultDNS = %v, want %v", result.DefaultDNS, "1.1.1.1")
	}
	if result.DefaultMTU != 1500 {
		t.Errorf("DefaultMTU = %v, want %v", result.DefaultMTU, 1500)
	}
	if result.PingURL != "https://cp.cloudflare.com" {
		t.Errorf("PingURL = %v, want %v", result.PingURL, "https://cp.cloudflare.com")
	}
}

func TestMergeConfig_NilCliOverrides(t *testing.T) {
	appCfg := &AppConfig{
		ServerPort: "8443",
		DefaultDNS: "1.1.1.1",
		DefaultMTU: 1500,
		PingURL:    "https://cp.cloudflare.com",
	}

	result := MergeConfig(appCfg, nil, nil)

	if result.ServerPort != "8443" {
		t.Errorf("ServerPort = %v, want %v", result.ServerPort, "8443")
	}
	if result.DefaultDNS != "1.1.1.1" {
		t.Errorf("DefaultDNS = %v, want %v", result.DefaultDNS, "1.1.1.1")
	}
	if result.DefaultMTU != 1500 {
		t.Errorf("DefaultMTU = %v, want %v", result.DefaultMTU, 1500)
	}
	if result.PingURL != "https://cp.cloudflare.com" {
		t.Errorf("PingURL = %v, want %v", result.PingURL, "https://cp.cloudflare.com")
	}
}

func TestMergeConfig_NilAppConfig(t *testing.T) {
	result := MergeConfig(nil, nil, nil)

	if result.ServerPort != "" {
		t.Errorf("ServerPort with nil appCfg = %v, want empty", result.ServerPort)
	}
	if result.DefaultDNS != "" {
		t.Errorf("DefaultDNS with nil appCfg = %v, want empty", result.DefaultDNS)
	}
}

func TestMergeConfig_NilAllParams(t *testing.T) {
	result := MergeConfig(nil, nil, nil)
	if result == nil {
		t.Error("MergeConfig(nil, nil, nil) returned nil, expected non-nil")
	}
}

func TestMergeConfig_PreserveRole(t *testing.T) {
	appCfg := &AppConfig{
		ServerPort: "8443",
	}

	runtimeCfg := &RuntimeCfg{
		Role:     "server",
		Password: "secret",
	}

	result := MergeConfig(appCfg, runtimeCfg, nil)

	if result.Role != "server" {
		t.Errorf("Role = %v, want %v (role should be preserved)", result.Role, "server")
	}
	if result.Password != "secret" {
		t.Errorf("Password = %v, want %v", result.Password, "secret")
	}
}

func TestSaveRuntimeCfg(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	cfg := &RuntimeCfg{
		Role:       "client",
		Password:   "testpass",
		ClientPeer: "example.com:8443",
		ServerPort: "9090",
	}

	err := SaveRuntimeCfg(cfgPath, cfg)
	if err != nil {
		t.Fatalf("SaveRuntimeCfg() error = %v", err)
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatalf("SaveRuntimeCfg() file not created: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	var loaded RuntimeCfg
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to parse saved config: %v", err)
	}

	if loaded.Role != "client" {
		t.Errorf("Loaded Role = %v, want %v", loaded.Role, "client")
	}
	if loaded.ServerPort != "9090" {
		t.Errorf("Loaded ServerPort = %v, want %v", loaded.ServerPort, "9090")
	}
}

func TestFindAppConfigPath(t *testing.T) {
	path := FindAppConfigPath()
	if path != "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("FindAppConfigPath() returned nonexistent path: %v", path)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"1500", 1500},
		{"0", 0},
		{"invalid", 0},
		{"", 0},
		{"-100", -100},
	}

	for _, tt := range tests {
		got := ParseInt(tt.input)
		if got != tt.want {
			t.Errorf("ParseInt(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestRuntimeCfg_Validate_ValidConfig(t *testing.T) {
	cfg := &RuntimeCfg{
		ServerPort: "8443",
		DefaultDNS: "1.1.1.1",
		DefaultMTU: 1500,
		PingURL:    "https://cp.cloudflare.com",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestRuntimeCfg_Validate_InvalidPort(t *testing.T) {
	tests := []string{"0", "65536", "abc", "-1", "99999"}
	for _, port := range tests {
		cfg := &RuntimeCfg{ServerPort: port}
		err := cfg.Validate()
		if err == nil {
			t.Errorf("Validate() expected error for port %q, got nil", port)
		}
	}
}

func TestRuntimeCfg_Validate_InvalidDNS(t *testing.T) {
	cfg := &RuntimeCfg{DefaultDNS: "not-an-ip"}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for invalid DNS, got nil")
	}
}

func TestRuntimeCfg_Validate_InvalidPingURL(t *testing.T) {
	cfg := &RuntimeCfg{PingURL: "://invalid"}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for invalid ping URL, got nil")
	}
}

func TestRuntimeCfg_Validate_InvalidMTU(t *testing.T) {
	tests := []int{0, 100, 575, 9001, 65535}
	for _, mtu := range tests {
		cfg := &RuntimeCfg{DefaultMTU: mtu}
		err := cfg.Validate()
		if err == nil {
			t.Errorf("Validate() expected error for MTU %d, got nil", mtu)
		}
	}
}
