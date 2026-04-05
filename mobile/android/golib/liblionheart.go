// Package golib provides Go bindings for Android with sing-box support
package golib

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lionheart-vpn/lionheart/core"
)

// VPNStatus represents the VPN connection status
type VPNStatus int

const (
	StatusDisconnected VPNStatus = iota
	StatusConnecting
	StatusConnected
	StatusReconnecting
	StatusError
)

func (s VPNStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusReconnecting:
		return "reconnecting"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Logger interface for Android callbacks
type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

// StatusCallback interface for Android status updates
type StatusCallback interface {
	OnStatus(status string)
	OnTurnInfo(url string)
	OnStats(tx, rx int64)
	OnLog(level, msg string)
}

// androidLogger wraps Android logger to implement core.Logger
type androidLogger struct {
	callback StatusCallback
}

func (l *androidLogger) Info(msg string)  { l.callback.OnLog("info", msg) }
func (l *androidLogger) Warn(msg string)  { l.callback.OnLog("warn", msg) }
func (l *androidLogger) Error(msg string) { l.callback.OnLog("error", msg) }

// androidStatusListener wraps Android callback to implement core.StatusListener
type androidStatusListener struct {
	callback StatusCallback
}

func (s *androidStatusListener) OnStatus(status string) { s.callback.OnStatus(status) }
func (s *androidStatusListener) OnTurnInfo(url string)  { s.callback.OnTurnInfo(url) }
func (s *androidStatusListener) OnStats(tx, rx int64)   { s.callback.OnStats(tx, rx) }

// LionheartVPN is the main VPN manager for Android
type LionheartVPN struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	
	smartKey     string
	password     string
	peer         string
	
	// Engine selection
	useSingBox   bool
	routingRules *core.RoutingRules
	
	// Legacy components
	session      *core.Session
	cache        *core.CredsCache
	reconnectCh  chan struct{}
	
	// Sing-box components
	singbox      *core.SingBoxEngine
	
	// Status
	status       VPNStatus
	txBytes      int64
	rxBytes      int64
	
	// Callbacks
	statusCallback StatusCallback
}

var (
	instance *LionheartVPN
	once     sync.Once
)

// GetInstance returns the singleton VPN instance
func GetInstance() *LionheartVPN {
	once.Do(func() {
		instance = &LionheartVPN{
			status: StatusDisconnected,
		}
	})
	return instance
}

// SetStatusCallback sets the Android status callback
func (v *LionheartVPN) SetStatusCallback(callback StatusCallback) {
	v.mu.Lock()
	v.statusCallback = callback
	v.mu.Unlock()
	
	// Set up core logger and listener
	if callback != nil {
		core.SetLogger(&androidLogger{callback: callback})
		core.SetListener(&androidStatusListener{callback: callback})
	}
}

// Configure configures the VPN with smart key
func (v *LionheartVPN) Configure(smartKey string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	v.smartKey = smartKey
	
	// Parse smart key
	peer, password, err := core.ParseSmartKey(smartKey)
	if err != nil {
		return fmt.Errorf("parse smart key: %w", err)
	}
	
	v.peer = peer
	v.password = password
	
	return nil
}

// ConfigureWithServer configures with server details directly
func (v *LionheartVPN) ConfigureWithServer(serverIP, port, password string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	v.peer = serverIP + ":" + port
	v.password = password
	v.smartKey = core.EncodeSmartKey(serverIP, port, password)
	
	return nil
}

// EnableSingBox enables/disables sing-box mode
func (v *LionheartVPN) EnableSingBox(enable bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.useSingBox = enable
}

// IsSingBoxEnabled returns true if sing-box mode is enabled
func (v *LionheartVPN) IsSingBoxEnabled() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.useSingBox
}

// SetRoutingRules sets routing rules from JSON string
func (v *LionheartVPN) SetRoutingRules(rulesJSON string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	if rulesJSON == "" {
		v.routingRules = nil
		return nil
	}
	
	var rules core.RoutingRules
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		return fmt.Errorf("parse routing rules: %w", err)
	}
	
	v.routingRules = &rules
	return nil
}

// SetRoutingPreset sets a predefined routing preset
func (v *LionheartVPN) SetRoutingPreset(presetName string) error {
	preset := core.GetPreset(presetName)
	if preset == nil {
		return fmt.Errorf("unknown preset: %s", presetName)
	}
	
	v.mu.Lock()
	v.routingRules = preset.Rules
	v.mu.Unlock()
	
	return nil
}

// GetAvailablePresets returns available routing presets as JSON
func (v *LionheartVPN) GetAvailablePresets() string {
	presets := core.GetPresetWithDescription()
	data, _ := json.Marshal(presets)
	return string(data)
}

// Connect starts the VPN connection
func (v *LionheartVPN) Connect() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	if v.status == StatusConnecting || v.status == StatusConnected {
		return fmt.Errorf("already connected or connecting")
	}
	
	if v.peer == "" {
		return fmt.Errorf("not configured")
	}
	
	v.ctx, v.cancel = context.WithCancel(context.Background())
	v.status = StatusConnecting
	
	if v.useSingBox {
		go v.connectSingBox()
	} else {
		go v.connectLegacy()
	}
	
	return nil
}

// connectLegacy establishes legacy KCP connection
func (v *LionheartVPN) connectLegacy() {
	v.cache = &core.CredsCache{}
	v.session = &core.Session{}
	v.reconnectCh = make(chan struct{}, 1)
	
	// Establish connection
	sess, closer, err := core.Establish(v.cache, v.peer, v.password, false)
	if err != nil {
		v.setStatus(StatusError)
		return
	}
	
	v.session.Set(sess, closer)
	v.setStatus(StatusConnected)
	
	// Start health check and reconnection loops
	go core.HealthLoop(v.ctx, v.session, v.reconnectCh)
	go core.ReconnectLoop(v.ctx, v.session, v.cache, v.peer, v.password, v.reconnectCh)
	
	// Monitor connection
	go v.monitorLegacyConnection()
}

// monitorLegacyConnection monitors the legacy connection status
func (v *LionheartVPN) monitorLegacyConnection() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-v.ctx.Done():
			return
		case <-ticker.C:
			v.mu.RLock()
			session := v.session
			v.mu.RUnlock()
			
			if session != nil {
				v.mu.Lock()
				v.txBytes = session.TxBytes.Load()
				v.rxBytes = session.RxBytes.Load()
				v.mu.Unlock()
			}
		}
	}
}

// connectSingBox establishes sing-box connection
func (v *LionheartVPN) connectSingBox() {
	// Parse server info
	host, portStr, err := net.SplitHostPort(v.peer)
	if err != nil {
		v.setStatus(StatusError)
		return
	}
	
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create sing-box configuration
	sbConfig := core.CreateDefaultConfig(v.smartKey, host, port, v.password, v.routingRules)
	
	// Create and initialize sing-box engine
	v.singbox = core.NewSingBoxEngine()
	if err := v.singbox.Initialize(sbConfig); err != nil {
		v.setStatus(StatusError)
		return
	}
	
	// Start sing-box
	if err := v.singbox.Start(); err != nil {
		v.setStatus(StatusError)
		return
	}
	
	v.setStatus(StatusConnected)
	
	// Monitor connection
	go v.monitorSingBoxConnection()
}

// monitorSingBoxConnection monitors sing-box connection status
func (v *LionheartVPN) monitorSingBoxConnection() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-v.ctx.Done():
			return
		case <-ticker.C:
			v.mu.RLock()
			singbox := v.singbox
			v.mu.RUnlock()
			
			if singbox != nil {
				status := singbox.Status()
				if status == "stopped" {
					v.setStatus(StatusError)
					return
				}
			}
		}
	}
}

// Disconnect stops the VPN connection
func (v *LionheartVPN) Disconnect() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	if v.cancel != nil {
		v.cancel()
	}
	
	if v.useSingBox && v.singbox != nil {
		v.singbox.Stop()
		v.singbox = nil
	} else if v.session != nil {
		v.session.Stop()
		v.session = nil
	}
	
	v.status = StatusDisconnected
	return nil
}

// GetStatus returns the current VPN status
func (v *LionheartVPN) GetStatus() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return int(v.status)
}

// GetStatusString returns the current VPN status as string
func (v *LionheartVPN) GetStatusString() string {
	return VPNStatus(v.GetStatus()).String()
}

func (v *LionheartVPN) setStatus(status VPNStatus) {
	v.mu.Lock()
	v.status = status
	v.mu.Unlock()
	
	if v.statusCallback != nil {
		v.statusCallback.OnStatus(status.String())
	}
}

// GetStats returns traffic statistics
func (v *LionheartVPN) GetStats() (tx, rx int64) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.txBytes, v.rxBytes
}

// GetSmartKey returns the configured smart key
func (v *LionheartVPN) GetSmartKey() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.smartKey
}

// GetServerIP returns the server IP
func (v *LionheartVPN) GetServerIP() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	host, _, err := net.SplitHostPort(v.peer)
	if err != nil {
		return ""
	}
	return host
}

// IsConfigured returns true if VPN is configured
func (v *LionheartVPN) IsConfigured() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.peer != ""
}

// GetVersion returns the core version
func (v *LionheartVPN) GetVersion() string {
	return core.Version
}

// ExportSingBoxConfig exports the current sing-box configuration as JSON
func (v *LionheartVPN) ExportSingBoxConfig() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	if v.peer == "" {
		return ""
	}
	
	host, portStr, err := net.SplitHostPort(v.peer)
	if err != nil {
		return ""
	}
	
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	
	sbConfig := core.CreateDefaultConfig(v.smartKey, host, port, v.password, v.routingRules)
	
	data, err := json.MarshalIndent(sbConfig, "", "  ")
	if err != nil {
		return ""
	}
	
	return string(data)
}

// ImportSingBoxConfig imports a sing-box configuration from JSON
func (v *LionheartVPN) ImportSingBoxConfig(configJSON string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	var config core.SingBoxConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	
	// Extract server info from config (if available)
	for _, outbound := range config.Outbounds {
		if outbound.Tag == "lionheart-out" {
			if settings, ok := outbound.Settings["server"].(string); ok {
				v.peer = settings
			}
			if port, ok := outbound.Settings["server_port"].(float64); ok {
				v.peer += fmt.Sprintf(":%d", int(port))
			}
			if pw, ok := outbound.Settings["password"].(string); ok {
				v.password = pw
			}
		}
	}
	
	v.useSingBox = true
	return nil
}
