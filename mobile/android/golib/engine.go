package golib

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lionheart-vpn/lionheart/core"
)

// Engine is a simplified VPN engine for mobile devices
type Engine struct {
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	
	// Configuration
	config     *EngineConfig
	
	// Components
	tunnel     *core.TunnelManager
	singbox    *core.SingBoxEngine
	
	// Status
	status     string
	connected  bool
	
	// Stats
	txBytes    int64
	rxBytes    int64
	
	// Callbacks
	onStatus   func(string)
	onStats    func(int64, int64)
	onLog      func(string)
}

// EngineConfig contains engine configuration
type EngineConfig struct {
	SmartKey     string
	Server       string
	Port         int
	Password     string
	UseSingBox   bool
	RoutingRules *core.RoutingRules
	MTU          int
	DNS          string
}

// DefaultEngineConfig returns default engine configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MTU:        9000,
		DNS:        "1.1.1.1",
		UseSingBox: true,
	}
}

// NewEngine creates a new VPN engine
func NewEngine(config *EngineConfig) *Engine {
	if config == nil {
		config = DefaultEngineConfig()
	}
	
	return &Engine{
		config: config,
		status: "disconnected",
	}
}

// SetCallbacks sets status callbacks
func (e *Engine) SetCallbacks(onStatus func(string), onStats func(int64, int64), onLog func(string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onStatus = onStatus
	e.onStats = onStats
	e.onLog = onLog
}

// Start starts the VPN engine
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.connected {
		return fmt.Errorf("already connected")
	}
	
	e.ctx, e.cancel = context.WithCancel(context.Background())
	e.status = "connecting"
	e.notifyStatus("connecting")
	
	// Create tunnel manager configuration
	tunnelConfig := &core.TunnelConfig{
		SmartKey:     e.config.SmartKey,
		Password:     e.config.Password,
		Peer:         fmt.Sprintf("%s:%d", e.config.Server, e.config.Port),
		UseSingBox:   e.config.UseSingBox,
		RoutingRules: e.config.RoutingRules,
	}
	
	// Create and start tunnel manager
	e.tunnel = core.NewTunnelManager(tunnelConfig)
	
	go func() {
		if err := e.tunnel.Start(); err != nil {
			e.log(fmt.Sprintf("Failed to start tunnel: %v", err))
			e.setStatus("error")
			return
		}
		e.setStatus("connected")
		e.monitorConnection()
	}()
	
	return nil
}

// Stop stops the VPN engine
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.cancel != nil {
		e.cancel()
	}
	
	if e.tunnel != nil {
		e.tunnel.Stop()
	}
	
	e.connected = false
	e.setStatus("disconnected")
	
	return nil
}

// Status returns the current status
func (e *Engine) Status() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

// IsConnected returns true if connected
func (e *Engine) IsConnected() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.connected
}

// GetStats returns traffic statistics
func (e *Engine) GetStats() (tx, rx int64) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.txBytes, e.rxBytes
}

func (e *Engine) setStatus(status string) {
	e.mu.Lock()
	e.status = status
	e.connected = (status == "connected")
	e.mu.Unlock()
	
	e.notifyStatus(status)
}

func (e *Engine) notifyStatus(status string) {
	if e.onStatus != nil {
		e.onStatus(status)
	}
}

func (e *Engine) notifyStats(tx, rx int64) {
	if e.onStats != nil {
		e.onStats(tx, rx)
	}
}

func (e *Engine) log(msg string) {
	if e.onLog != nil {
		e.onLog(msg)
	}
}

func (e *Engine) monitorConnection() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			if e.tunnel != nil {
				tx, rx := e.tunnel.GetStats()
				e.mu.Lock()
				e.txBytes = tx
				e.rxBytes = rx
				e.mu.Unlock()
				e.notifyStats(tx, rx)
				
				// Check status
				status := e.tunnel.Status()
				if status != e.status {
					e.setStatus(status)
				}
			}
		}
	}
}

// GetTUNConfig returns TUN interface configuration for Android
func (e *Engine) GetTUNConfig() (*TUNConfig, error) {
	if !e.connected {
		return nil, fmt.Errorf("not connected")
	}
	
	return &TUNConfig{
		Address:    "172.19.0.2/30",
		Gateway:    "172.19.0.1",
		DNS:        e.config.DNS,
		MTU:        e.config.MTU,
		IPv6Address: "fdfe:dcba:9876::2/126",
		IPv6Gateway: "fdfe:dcba:9876::1",
	}, nil
}

// TUNConfig contains TUN interface configuration
type TUNConfig struct {
	Address     string
	Gateway     string
	DNS         string
	MTU         int
	IPv6Address string
	IPv6Gateway string
}

// CreateDefaultRoute creates default route configuration
func (e *Engine) CreateDefaultRoute() *RouteConfig {
	return &RouteConfig{
		Destination: "0.0.0.0/0",
		Gateway:     "172.19.0.1",
		Interface:   "tun0",
	}
}

// RouteConfig contains route configuration
type RouteConfig struct {
	Destination string
	Gateway     string
	Interface   string
	Metric      int
}

// GetSingBoxConfig returns the sing-box configuration (if using sing-box)
func (e *Engine) GetSingBoxConfig() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if e.tunnel == nil || !e.tunnel.IsSingBox() {
		return ""
	}
	
	engine := e.tunnel.GetSingBoxEngine()
	if engine == nil {
		return ""
	}
	
	// Export configuration
	// This would need to be implemented in the core package
	return ""
}

// SetRoutingRules updates routing rules dynamically
func (e *Engine) SetRoutingRules(rules *core.RoutingRules) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.config.RoutingRules = rules
	
	// Update tunnel manager if running
	if e.tunnel != nil {
		e.tunnel.SetRoutingRules(rules)
	}
	
	return nil
}

// EnableSingBox enables/disables sing-box mode
func (e *Engine) EnableSingBox(enable bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.config.UseSingBox = enable
	
	if e.tunnel != nil {
		e.tunnel.EnableSingBox(enable)
	}
}

// QuickConnect is a simplified connect method for Android
func QuickConnect(smartKey string, callback StatusCallback) (*Engine, error) {
	// Parse smart key
	peer, password, err := core.ParseSmartKey(smartKey)
	if err != nil {
		return nil, fmt.Errorf("parse smart key: %w", err)
	}
	
	// Parse server and port
	host, portStr, err := net.SplitHostPort(peer)
	if err != nil {
		return nil, fmt.Errorf("parse peer: %w", err)
	}
	
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create engine config
	config := &EngineConfig{
		SmartKey:   smartKey,
		Server:     host,
		Port:       port,
		Password:   password,
		UseSingBox: true,
		RoutingRules: &core.RoutingRules{
			GeoSiteBlock: []string{"category-ads-all"},
			Final:        "proxy",
		},
	}
	
	// Create engine
	engine := NewEngine(config)
	
	// Set callbacks
	engine.SetCallbacks(
		func(status string) {
			callback.OnStatus(status)
		},
		func(tx, rx int64) {
			callback.OnStats(tx, rx)
		},
		func(msg string) {
			callback.OnLog("info", msg)
		},
	)
	
	// Start engine
	if err := engine.Start(); err != nil {
		return nil, err
	}
	
	return engine, nil
}
