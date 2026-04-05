// Package core provides sing-box transport integration for Lionheart
package core

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/xtaci/kcp-go/v5"
)

// LionheartTransport implements a custom sing-box outbound transport
// using KCP + Yamux + TURN relay (preserving original Lionheart protocol)
type LionheartTransport struct {
	ctx      context.Context
	cancel   context.CancelFunc
	peer     string
	password string
	session  *yamux.Session
	closer   io.Closer
	cache    *CredsCache
	mu       sync.RWMutex
	
	// Callbacks
	onStatusChange func(string)
	onTurnInfo     func(string)
	onStats        func(int64, int64)
	
	// Stats
	txBytes int64
	rxBytes int64
}

// TransportConfig contains configuration for Lionheart transport
type TransportConfig struct {
	Server   string `json:"server"`
	Port     int    `json:"server_port"`
	Password string `json:"password"`
	SmartKey string `json:"smart_key,omitempty"`
	
	// KCP tuning
	DataShards   int `json:"data_shards,omitempty"`
	ParityShards int `json:"parity_shards,omitempty"`
	WindowSize   int `json:"window_size,omitempty"`
	
	// Reconnection
	MaxRetries    int           `json:"max_retries,omitempty"`
	RetryInterval time.Duration `json:"retry_interval,omitempty"`
	MaxBackoff    time.Duration `json:"max_backoff,omitempty"`
}

// DefaultTransportConfig returns default transport configuration
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{
		DataShards:    10,
		ParityShards:  3,
		WindowSize:    1024,
		MaxRetries:    10,
		RetryInterval: 2 * time.Second,
		MaxBackoff:    60 * time.Second,
	}
}

// NewLionheartTransport creates a new Lionheart transport
func NewLionheartTransport(config *TransportConfig) *LionheartTransport {
	ctx, cancel := context.WithCancel(context.Background())
	
	peer := fmt.Sprintf("%s:%d", config.Server, config.Port)
	if config.SmartKey != "" {
		if p, pw, err := ParseSmartKey(config.SmartKey); err == nil {
			peer = p
			if config.Password == "" {
				config.Password = pw
			}
		}
	}

	return &LionheartTransport{
		ctx:      ctx,
		cancel:   cancel,
		peer:     peer,
		password: config.Password,
		cache:    &CredsCache{},
	}
}

// SetCallbacks sets status callbacks
func (t *LionheartTransport) SetCallbacks(onStatus func(string), onTurn func(string), onStats func(int64, int64)) {
	t.onStatusChange = onStatus
	t.onTurnInfo = onTurn
	t.onStats = onStats
}

// Connect establishes connection to the server
func (t *LionheartTransport) Connect() error {
	if t.onStatusChange != nil {
		t.onStatusChange("connecting")
	}

	sess, closer, err := Establish(t.cache, t.peer, t.password, false)
	if err != nil {
		if t.onStatusChange != nil {
			t.onStatusChange("error")
		}
		return fmt.Errorf("establish connection: %w", err)
	}

	t.mu.Lock()
	t.session = sess
	t.closer = closer
	t.mu.Unlock()

	if t.onStatusChange != nil {
		t.onStatusChange("connected")
	}

	// Start health check and reconnection loops
	reconnectCh := make(chan struct{}, 1)
	go t.healthLoop(reconnectCh)
	go t.reconnectLoop(reconnectCh)

	return nil
}

// Close closes the transport
func (t *LionheartTransport) Close() error {
	t.cancel()
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closer != nil {
		t.closer.Close()
		t.closer = nil
	}
	t.session = nil
	
	if t.onStatusChange != nil {
		t.onStatusChange("disconnected")
	}
	
	return nil
}

// Dial creates a new connection through the transport
func (t *LionheartTransport) Dial(network, address string) (net.Conn, error) {
	t.mu.RLock()
	session := t.session
	t.mu.RUnlock()

	if session == nil {
		return nil, fmt.Errorf("not connected")
	}

	stream, err := session.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}

	// Wrap stream with stats tracking
	return &statsConn{
		Conn:   stream,
		onRead: t.trackRx,
		onWrite: t.trackTx,
	}, nil
}

// IsConnected returns true if transport is connected
func (t *LionheartTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.session == nil {
		return false
	}
	
	// Try to ping
	_, err := t.session.Ping()
	return err == nil
}

// GetStats returns current traffic statistics
func (t *LionheartTransport) GetStats() (tx, rx int64) {
	return t.txBytes, t.rxBytes
}

func (t *LionheartTransport) trackTx(n int) {
	t.txBytes += int64(n)
	if t.onStats != nil {
		t.onStats(t.txBytes, t.rxBytes)
	}
}

func (t *LionheartTransport) trackRx(n int) {
	t.rxBytes += int64(n)
	if t.onStats != nil {
		t.onStats(t.txBytes, t.rxBytes)
	}
}

func (t *LionheartTransport) healthLoop(reconnectCh chan<- struct{}) {
	ticker := time.NewTicker(HealthEvery)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.mu.RLock()
			session := t.session
			t.mu.RUnlock()

			if session != nil {
				if _, err := session.Ping(); err != nil {
					getLog().Warn("Connection lost, triggering reconnect")
					t.mu.Lock()
					t.session = nil
					t.mu.Unlock()
					
					select {
					case reconnectCh <- struct{}{}:
					default:
					}
				}
			}
		}
	}
}

func (t *LionheartTransport) reconnectLoop(reconnectCh <-chan struct{}) {
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-reconnectCh:
			if t.onStatusChange != nil {
				t.onStatusChange("reconnecting")
			}

			backoff := 2 * time.Second
			attempt := 1

			for {
				select {
				case <-t.ctx.Done():
					return
				default:
				}

				getLog().Info(fmt.Sprintf("Reconnecting (attempt %d)...", attempt))
				
				forceRefresh := attempt > 3
				sess, closer, err := Establish(t.cache, t.peer, t.password, forceRefresh)
				
				if err == nil {
					t.mu.Lock()
					t.session = sess
					t.closer = closer
					t.mu.Unlock()
					
					if t.onStatusChange != nil {
						t.onStatusChange("connected")
					}
					getLog().Info("Connection restored")
					break
				}

				getLog().Warn(fmt.Sprintf("Reconnect attempt %d failed: %v", attempt, err))
				
				select {
				case <-t.ctx.Done():
					return
				case <-time.After(backoff):
				}

				backoff *= 2
				if backoff > MaxBackoff {
					backoff = MaxBackoff
				}
				attempt++
			}
		}
	}
}

// statsConn wraps a net.Conn to track traffic statistics
type statsConn struct {
	net.Conn
	onRead  func(int)
	onWrite func(int)
}

func (c *statsConn) Read(p []byte) (n int, err error) {
	n, err = c.Conn.Read(p)
	if n > 0 && c.onRead != nil {
		c.onRead(n)
	}
	return
}

func (c *statsConn) Write(p []byte) (n int, err error) {
	n, err = c.Conn.Write(p)
	if n > 0 && c.onWrite != nil {
		c.onWrite(n)
	}
	return
}

// LionheartOutbound is a sing-box compatible outbound adapter
type LionheartOutbound struct {
	ctx       context.Context
	transport *LionheartTransport
	config    *TransportConfig
}

// NewLionheartOutbound creates a new Lionheart outbound adapter
func NewLionheartOutbound(ctx context.Context, config *TransportConfig) (*LionheartOutbound, error) {
	if config == nil {
		config = DefaultTransportConfig()
	}

	transport := NewLionheartTransport(config)
	
	return &LionheartOutbound{
		ctx:       ctx,
		transport: transport,
		config:    config,
	}, nil
}

// Type returns the outbound type
func (o *LionheartOutbound) Type() string {
	return "lionheart"
}

// Tag returns the outbound tag
func (o *LionheartOutbound) Tag() string {
	return "lionheart-out"
}

// Start starts the outbound
func (o *LionheartOutbound) Start() error {
	return o.transport.Connect()
}

// Close closes the outbound
func (o *LionheartOutbound) Close() error {
	return o.transport.Close()
}

// DialContext dials a connection
func (o *LionheartOutbound) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return o.transport.Dial(network, address)
}

// NewConnection implements the adapter.Outbound interface for sing-box
func (o *LionheartOutbound) NewConnection(ctx context.Context, conn net.Conn, metadata adapter.Metadata) error {
	// Dial upstream through Lionheart transport
	upstream, err := o.DialContext(ctx, metadata.Network, metadata.Destination.String())
	if err != nil {
		return err
	}
	defer upstream.Close()

	// Relay traffic
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(upstream, conn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(conn, upstream)
	}()

	wg.Wait()
	return nil
}

// NewPacketConnection implements the adapter.Outbound interface for sing-box (UDP)
func (o *LionheartOutbound) NewPacketConnection(ctx context.Context, conn net.PacketConn, metadata adapter.Metadata) error {
	// For UDP, we need to handle it differently
	// This is a simplified implementation
	return fmt.Errorf("UDP not yet implemented in Lionheart transport")
}

// RegisterOutbound registers the Lionheart outbound with sing-box
func RegisterOutbound(registry *adapter.OutboundRegistry) {
	// This would be called during sing-box initialization
	// to register the custom Lionheart outbound type
}

// CreateOutbound creates a Lionheart outbound from configuration
func CreateOutbound(ctx context.Context, router adapter.Router, logger log.Logger, tag string, options option.Outbound) (adapter.Outbound, error) {
	// Parse options
	config := &TransportConfig{}
	
	if options.LionheartOptions != nil {
		// Parse from sing-box options
		if server, ok := options.LionheartOptions["server"].(string); ok {
			config.Server = server
		}
		if port, ok := options.LionheartOptions["server_port"].(float64); ok {
			config.Port = int(port)
		}
		if password, ok := options.LionheartOptions["password"].(string); ok {
			config.Password = password
		}
		if smartKey, ok := options.LionheartOptions["smart_key"].(string); ok {
			config.SmartKey = smartKey
		}
	}

	return NewLionheartOutbound(ctx, config)
}
