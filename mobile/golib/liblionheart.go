// Package golib is the gomobile bridge between Android/iOS and the core engine.
// It's intentionally thin — all tunnel logic lives in core/.
package golib

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lionheart-vpn/lionheart/core"
)

// EventListener is the interface the Kotlin/Swift side implements.
type EventListener interface {
	OnStatusChanged(status string)
	OnLog(level, message string)
	OnStatsUpdate(tx, rx int64)
	OnTurnInfo(url string)
}

// --- bridge logger → core.Logger ---

type mobileLogger struct{}

func (mobileLogger) Info(msg string)  { logToApp("info", msg) }
func (mobileLogger) Warn(msg string)  { logToApp("warn", msg) }
func (mobileLogger) Error(msg string) { logToApp("error", msg) }

// --- bridge status → core.StatusListener ---

type mobileStatus struct{}

func (mobileStatus) OnStatus(s string)         { emit(s) }
func (mobileStatus) OnTurnInfo(url string)      { appLis().OnTurnInfo(url) }
func (mobileStatus) OnStats(tx, rx int64)       { appLis().OnStatsUpdate(tx, rx) }

// --- globals ---

var (
	mu       sync.Mutex
	vpnEng   *vpnEngine
	listener EventListener
	runDone  chan struct{}
)

func appLis() EventListener {
	mu.Lock()
	defer mu.Unlock()
	if listener == nil {
		return nopListener{}
	}
	return listener
}

type nopListener struct{}
func (nopListener) OnStatusChanged(string)     {}
func (nopListener) OnLog(string, string)       {}
func (nopListener) OnStatsUpdate(int64, int64) {}
func (nopListener) OnTurnInfo(string)          {}

// --- Public API (exported to Kotlin/Swift via gomobile) ---

func SetListener(l EventListener) {
	mu.Lock()
	listener = l
	mu.Unlock()
	// Wire up core loggers
	core.SetLogger(mobileLogger{})
	core.SetListener(mobileStatus{})
}

func Start(smartKey string, tunFd int, mtu int, dns string) error {
	mu.Lock()
	if vpnEng != nil {
		mu.Unlock()
		return fmt.Errorf("already running")
	}
	peer, pw, err := core.ParseSmartKey(smartKey)
	if err != nil {
		mu.Unlock()
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := &vpnEngine{
		ctx:    ctx,
		cancel: cancel,
		peer:   peer,
		pw:     pw,
		tunFd:  tunFd,
		mtu:    mtu,
		dns:    dns,
	}
	vpnEng = e
	runDone = make(chan struct{})
	mu.Unlock()

	emit("connecting")
	logToApp("info", "[LIB] Start() → run()...")
	err = e.run()
	logToApp("info", fmt.Sprintf("[LIB] Start() → run() returned err=%v", err))
	close(runDone)
	return err
}

func Stop() {
	logToApp("info", "[LIB] Stop() called")
	mu.Lock()
	e := vpnEng
	done := runDone
	vpnEng = nil
	mu.Unlock()
	if e != nil {
		logToApp("info", "[LIB] Stop() → cancel + closeAll...")
		e.stop()
		if done != nil {
			select {
			case <-done:
				logToApp("info", "[LIB] Stop() → run() finished cleanly")
			case <-time.After(5 * time.Second):
				logToApp("warn", "[LIB] Stop() → timeout waiting for run()")
			}
		}
	}
	emit("disconnected")
}

func IsRunning() bool {
	mu.Lock()
	defer mu.Unlock()
	return vpnEng != nil
}

func GetStatus() string {
	mu.Lock()
	defer mu.Unlock()
	if vpnEng == nil {
		return "disconnected"
	}
	return vpnEng.status
}

func ParseSmartKeyInfo(smartKey string) (string, error) {
	return core.SmartKeyServerIP(smartKey)
}

func GetVersion() string {
	return core.Version
}

// --- helpers ---

func emit(status string) {
	mu.Lock()
	l := listener
	if vpnEng != nil {
		vpnEng.status = status
	}
	mu.Unlock()
	if l != nil {
		l.OnStatusChanged(status)
	}
}

func logToApp(level, msg string) {
	mu.Lock()
	l := listener
	mu.Unlock()
	if l != nil {
		l.OnLog(level, msg)
	}
}
