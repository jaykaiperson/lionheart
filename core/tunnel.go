// Package core is the shared Lionheart tunnel engine with sing-box integration.
// Both the CLI (cmd/lionheart) and mobile bridge (mobile/golib) import this.
package core

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/pion/turn/v4"
	"github.com/xtaci/kcp-go/v5"
)

const (
	Version     = "1.4" // Updated for sing-box integration
	DefPort     = "8443"
	MaxBackoff  = 60 * time.Second
	HealthEvery = 15 * time.Second
	CredsTTL    = 5 * time.Minute
)

// Logger is the interface both CLI and mobile provide for log output.
type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

// StatusListener receives tunnel state changes.
type StatusListener interface {
	OnStatus(status string)
	OnTurnInfo(url string)
	OnStats(tx, rx int64)
}

// --- default no-op implementations ---

type nopLogger struct{}

func (nopLogger) Info(string)  {}
func (nopLogger) Warn(string)  {}
func (nopLogger) Error(string) {}

type nopStatus struct{}

func (nopStatus) OnStatus(string)     {}
func (nopStatus) OnTurnInfo(string)   {}
func (nopStatus) OnStats(int64, int64) {}

// --- globals set by the host ---

var (
	mu  sync.Mutex
	Log Logger         = nopLogger{}
	Lis StatusListener = nopStatus{}
)

func SetLogger(l Logger)         { mu.Lock(); Log = l; mu.Unlock() }
func SetListener(l StatusListener) { mu.Lock(); Lis = l; mu.Unlock() }

func getLog() Logger           { mu.Lock(); defer mu.Unlock(); return Log }
func getLis() StatusListener   { mu.Lock(); defer mu.Unlock(); return Lis }

// --- Key derivation ---

func DeriveKey(pw string) []byte { h := sha256.Sum256([]byte(pw)); return h[:] }

// --- Smart-key ---

func ParseSmartKey(k string) (peer, pw string, err error) {
	var d []byte
	d, err = base64.RawURLEncoding.DecodeString(k)
	if err != nil {
		d, err = base64.RawStdEncoding.DecodeString(k)
		if err != nil {
			return "", "", fmt.Errorf("invalid smart-key")
		}
	}
	parts := strings.SplitN(string(d), "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("corrupted smart-key")
	}
	peer = parts[0]
	if !strings.Contains(peer, ":") {
		peer += ":" + DefPort
	}
	pw = parts[1]
	return
}

func EncodeSmartKey(ip, port, password string) string {
	raw := ip + ":" + port + "|" + password
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func SmartKeyServerIP(k string) (string, error) {
	peer, _, err := ParseSmartKey(k)
	if err != nil {
		return "", err
	}
	host, _, _ := net.SplitHostPort(peer)
	return host, nil
}

// --- Yamux config ---

func YmxCfg() *yamux.Config {
	c := yamux.DefaultConfig()
	c.EnableKeepAlive = true
	c.KeepAliveInterval = 10 * time.Second
	c.ConnectionWriteTimeout = 10 * time.Second
	c.StreamOpenTimeout = 10 * time.Second
	return c
}

// --- Closer helpers ---

type CloserFunc func()

func (f CloserFunc) Close() error { f(); return nil }

type MultiCloser struct{ CC []io.Closer }

func (m *MultiCloser) Close() error {
	for _, c := range m.CC {
		c.Close()
	}
	return nil
}

// --- TURN dial ---

func DialTURN(cred TurnCred, peer, pw string) (*yamux.Session, io.Closer, error) {
	addr := strings.TrimPrefix(strings.TrimPrefix(strings.Split(cred.URL, "?")[0], "turn:"), "turns:")
	uc, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, nil, err
	}
	tc, err := turn.NewClient(&turn.ClientConfig{
		STUNServerAddr: addr,
		TURNServerAddr: addr,
		Conn:           uc,
		Username:       cred.User,
		Password:       cred.Pass,
	})
	if err != nil {
		uc.Close()
		return nil, nil, err
	}
	if err := tc.Listen(); err != nil {
		tc.Close()
		uc.Close()
		return nil, nil, err
	}
	relay, err := tc.Allocate()
	if err != nil {
		tc.Close()
		uc.Close()
		return nil, nil, err
	}
	blk, _ := kcp.NewAESBlockCrypt(DeriveKey(pw))
	kc, err := kcp.NewConn(peer, blk, 10, 3, relay)
	if err != nil {
		tc.Close()
		uc.Close()
		return nil, nil, err
	}
	kc.SetNoDelay(1, 10, 2, 1)
	kc.SetWindowSize(1024, 1024)
	kc.SetStreamMode(true)
	ym, err := yamux.Client(kc, YmxCfg())
	if err != nil {
		kc.Close()
		tc.Close()
		uc.Close()
		return nil, nil, err
	}
	return ym, &MultiCloser{[]io.Closer{ym, kc, CloserFunc(func() { tc.Close() }), uc}}, nil
}

// --- Establish tunnel (try all TURN servers) ---

func Establish(cache *CredsCache, peer, pw string, force bool) (*yamux.Session, io.Closer, error) {
	creds, err := cache.Get(force)
	if err != nil {
		return nil, nil, err
	}
	var turnCreds []TurnCred
	for _, c := range creds {
		if strings.HasPrefix(c.URL, "turn") {
			turnCreds = append(turnCreds, c)
		}
	}
	if len(turnCreds) == 0 {
		return nil, nil, fmt.Errorf("no TURN servers found")
	}
	log := getLog()
	lis := getLis()
	var lastErr error
	for i, c := range turnCreds {
		log.Info(fmt.Sprintf("TURN %d/%d: %s", i+1, len(turnCreds), c.URL))
		lis.OnTurnInfo(c.URL)
		ym, cl, err := DialTURN(c, peer, pw)
		if err != nil {
			log.Warn(fmt.Sprintf("TURN failed: %v", err))
			lastErr = err
			continue
		}
		ch := make(chan error, 1)
		go func() { _, e := ym.Ping(); ch <- e }()
		select {
		case e := <-ch:
			if e == nil {
				return ym, cl, nil
			}
			lastErr = e
		case <-time.After(5 * time.Second):
			lastErr = fmt.Errorf("ping timeout")
		}
		cl.Close()
		log.Warn(fmt.Sprintf("  %v", lastErr))
	}
	return nil, nil, fmt.Errorf("all TURN servers unreachable: %v", lastErr)
}

// --- Session wrapper ---

type Session struct {
	sync.RWMutex
	Ym *yamux.Session
	Cl io.Closer
	Ok bool
	TxBytes atomic.Int64
	RxBytes atomic.Int64
}

func (s *Session) Set(y *yamux.Session, c io.Closer) {
	s.Lock()
	defer s.Unlock()
	if s.Cl != nil {
		s.Cl.Close()
	}
	s.Ym, s.Cl, s.Ok = y, c, true
}

func (s *Session) Get() (*yamux.Session, bool) {
	s.RLock()
	defer s.RUnlock()
	return s.Ym, s.Ok
}

func (s *Session) Down() {
	s.Lock()
	defer s.Unlock()
	s.Ok = false
}

func (s *Session) Stop() {
	s.Lock()
	defer s.Unlock()
	if s.Cl != nil {
		s.Cl.Close()
		s.Cl = nil
	}
	s.Ym, s.Ok = nil, false
}

// --- Health + reconnect loops ---

func HealthLoop(ctx context.Context, sess *Session, rch chan<- struct{}) {
	log := getLog()
	tk := time.NewTicker(HealthEvery)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			if y, ok := sess.Get(); ok && y != nil {
				if _, e := y.Ping(); e != nil {
					log.Warn("Connection lost")
					sess.Down()
					select {
					case rch <- struct{}{}:
					default:
					}
				}
			}
		}
	}
}

func ReconnectLoop(ctx context.Context, sess *Session, cache *CredsCache, peer, pw string, rch <-chan struct{}) {
	log := getLog()
	lis := getLis()
	for {
		select {
		case <-ctx.Done():
			return
		case <-rch:
			lis.OnStatus("reconnecting")
			bo := 2 * time.Second
			for a := 1; ; a++ {
				select {
				case <-ctx.Done():
					return
				default:
				}
				log.Info(fmt.Sprintf("Reconnecting (#%d)...", a))
				y, c, e := Establish(cache, peer, pw, a > 3)
				if e == nil {
					sess.Set(y, c)
					lis.OnStatus("connected")
					log.Info("Connection restored!")
					break
				}
				log.Warn(fmt.Sprintf("Attempt %d failed: %v", a, e))
				select {
				case <-ctx.Done():
					return
				case <-time.After(bo):
				}
				if bo *= 2; bo > MaxBackoff {
					bo = MaxBackoff
				}
			}
		}
	}
}

// --- Tunnel Manager with sing-box integration ---

// TunnelManager manages both legacy KCP tunnel and sing-box engine
type TunnelManager struct {
	ctx         context.Context
	cancel      context.CancelFunc
	smartKey    string
	password    string
	peer        string
	
	// Legacy components
	session     *Session
	cache       *CredsCache
	reconnectCh chan struct{}
	
	// Sing-box components
	singbox     *SingBoxEngine
	useSingBox  bool
	routingRules *RoutingRules
	
	// Status
	mu          sync.RWMutex
	status      string
	txBytes     int64
	rxBytes     int64
}

// TunnelConfig contains tunnel configuration
type TunnelConfig struct {
	SmartKey     string
	Password     string
	Peer         string
	UseSingBox   bool
	RoutingRules *RoutingRules
	SingBoxConfig *SingBoxConfig
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(config *TunnelConfig) *TunnelManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	peer := config.Peer
	password := config.Password
	
	// Parse smart key if provided
	if config.SmartKey != "" {
		if p, pw, err := ParseSmartKey(config.SmartKey); err == nil {
			peer = p
			if password == "" {
				password = pw
			}
		}
	}
	
	return &TunnelManager{
		ctx:         ctx,
		cancel:      cancel,
		smartKey:    config.SmartKey,
		password:    password,
		peer:        peer,
		session:     &Session{},
		cache:       &CredsCache{},
		reconnectCh: make(chan struct{}, 1),
		useSingBox:  config.UseSingBox,
		routingRules: config.RoutingRules,
		status:      "disconnected",
	}
}

// Start starts the tunnel
func (tm *TunnelManager) Start() error {
	if tm.useSingBox {
		return tm.startSingBox()
	}
	return tm.startLegacy()
}

// startLegacy starts the legacy KCP tunnel
func (tm *TunnelManager) startLegacy() error {
	tm.setStatus("connecting")
	
	// Establish initial connection
	sess, closer, err := Establish(tm.cache, tm.peer, tm.password, false)
	if err != nil {
		tm.setStatus("error")
		return fmt.Errorf("establish tunnel: %w", err)
	}
	
	tm.session.Set(sess, closer)
	tm.setStatus("connected")
	
	// Start health check and reconnection loops
	go HealthLoop(tm.ctx, tm.session, tm.reconnectCh)
	go ReconnectLoop(tm.ctx, tm.session, tm.cache, tm.peer, tm.password, tm.reconnectCh)
	
	return nil
}

// startSingBox starts the sing-box engine
func (tm *TunnelManager) startSingBox() error {
	tm.setStatus("connecting")
	
	// Parse server info from peer
	host, portStr, err := net.SplitHostPort(tm.peer)
	if err != nil {
		return fmt.Errorf("parse peer address: %w", err)
	}
	
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create sing-box configuration
	var sbConfig *SingBoxConfig
	if tm.singbox != nil && tm.singbox.config != nil {
		sbConfig = tm.singbox.config
	} else {
		sbConfig = CreateDefaultConfig(tm.smartKey, host, port, tm.password, tm.routingRules)
	}
	
	// Create and initialize sing-box engine
	tm.singbox = NewSingBoxEngine()
	if err := tm.singbox.Initialize(sbConfig); err != nil {
		tm.setStatus("error")
		return fmt.Errorf("initialize sing-box: %w", err)
	}
	
	// Start sing-box
	if err := tm.singbox.Start(); err != nil {
		tm.setStatus("error")
		return fmt.Errorf("start sing-box: %w", err)
	}
	
	tm.setStatus("connected")
	return nil
}

// Stop stops the tunnel
func (tm *TunnelManager) Stop() error {
	tm.cancel()
	
	if tm.useSingBox && tm.singbox != nil {
		tm.singbox.Stop()
	} else {
		tm.session.Stop()
	}
	
	tm.setStatus("disconnected")
	return nil
}

// Status returns current tunnel status
func (tm *TunnelManager) Status() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.status
}

func (tm *TunnelManager) setStatus(status string) {
	tm.mu.Lock()
	tm.status = status
	tm.mu.Unlock()
	
	lis := getLis()
	lis.OnStatus(status)
}

// GetStats returns traffic statistics
func (tm *TunnelManager) GetStats() (tx, rx int64) {
	if tm.useSingBox {
		// For sing-box, stats would come from the engine
		return tm.txBytes, tm.rxBytes
	}
	return tm.session.TxBytes.Load(), tm.session.RxBytes.Load()
}

// IsSingBox returns true if using sing-box engine
func (tm *TunnelManager) IsSingBox() bool {
	return tm.useSingBox
}

// GetSingBoxEngine returns the sing-box engine (if using sing-box)
func (tm *TunnelManager) GetSingBoxEngine() *SingBoxEngine {
	return tm.singbox
}

// EnableSingBox enables/disables sing-box mode
func (tm *TunnelManager) EnableSingBox(enable bool) {
	tm.useSingBox = enable
}

// SetRoutingRules sets routing rules for sing-box
func (tm *TunnelManager) SetRoutingRules(rules *RoutingRules) {
	tm.routingRules = rules
}
