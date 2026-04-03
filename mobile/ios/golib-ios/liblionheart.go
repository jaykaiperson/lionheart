package golib

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type EventListener interface {
	OnStatusChanged(status string)
	OnLog(level, message string)
	OnStatsUpdate(tx, rx int64)
	OnTurnInfo(url string)
}

var (
	mu       sync.Mutex
	vpnEng   *vpnEngine
	listener EventListener
	runDone  chan struct{}
)

func appLis() EventListener {
	mu.Lock()
	defer mu.Unlock()
	if listener != nil { return listener }
	return nopListener{}
}

type nopListener struct{}
func (nopListener) OnStatusChanged(string)      {}
func (nopListener) OnLog(string, string)         {}
func (nopListener) OnStatsUpdate(int64, int64)   {}
func (nopListener) OnTurnInfo(string)            {}

func SetListener(l EventListener) { mu.Lock(); defer mu.Unlock(); listener = l }

func Start(smartKey string, tunFd int, mtu int, dns string) error {
	mu.Lock()
	if vpnEng != nil { mu.Unlock(); return fmt.Errorf("already running") }
	peer, pw, err := decodeSmartKey(smartKey)
	if err != nil { mu.Unlock(); return fmt.Errorf("bad smart key: %w", err) }
	ctx, cancel := context.WithCancel(context.Background())
	eng := &vpnEngine{ctx: ctx, cancel: cancel, peer: peer, pw: pw, tunFd: tunFd, mtu: mtu, dns: dns, status: "disconnected"}
	vpnEng = eng
	runDone = make(chan struct{})
	mu.Unlock()
	go func() { defer close(runDone); if err := eng.run(); err != nil { logToApp("ERROR", err.Error()) } }()
	return nil
}

func Stop() {
	mu.Lock(); eng := vpnEng; done := runDone; mu.Unlock()
	if eng == nil { return }
	eng.stop()
	if done != nil { select { case <-done: case <-time.After(5 * time.Second): } }
	mu.Lock(); vpnEng = nil; mu.Unlock()
}

func IsRunning() bool { mu.Lock(); defer mu.Unlock(); return vpnEng != nil }

func GetStatus() string {
	mu.Lock(); defer mu.Unlock()
	if vpnEng == nil { return "disconnected" }
	return vpnEng.status
}

func ParseSmartKeyInfo(smartKey string) (string, error) {
	peer, _, err := decodeSmartKey(smartKey)
	return peer, err
}

func GetVersion() string { return versionStr }

func emit(status string) { l := appLis(); l.OnStatusChanged(status); l.OnLog("INFO", "Status: "+status) }
func logToApp(level, msg string) { l := appLis(); l.OnLog(level, msg) }
