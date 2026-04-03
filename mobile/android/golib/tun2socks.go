package golib

import (
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

type tun2Socks struct {
	mu      sync.Mutex
	running bool
	nfd     int
}

func newTun2Socks(tunFd int, socksAddr string, mtu int, dns string) (*tun2Socks, error) {
	if mtu <= 0 {
		mtu = 1500
	}

	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(tunFd), syscall.F_GETFL, 0)
	if errno != 0 {
		return nil, fmt.Errorf("[T2S] fd %d invalid: %v", tunFd, errno)
	}

	r1, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(tunFd), syscall.F_DUPFD, 512)
	if errno != 0 {
		return nil, fmt.Errorf("[T2S] syscall.F_DUPFD failed: %v", errno)
	}
	nfd := int(r1)

	logToApp("info", fmt.Sprintf("[T2S] init: orig_fd=%d safe_dup_fd=%d socks=%s mtu=%d dns=%s", tunFd, nfd, socksAddr, mtu, dns))

	key := &engine.Key{
		Proxy:                fmt.Sprintf("socks5://%s", socksAddr),
		Device:               fmt.Sprintf("fd://%d", nfd),
		MTU:                  mtu,
		LogLevel:             "debug",
		TCPSendBufferSize:    "128kb",
		TCPReceiveBufferSize: "128kb",
	}
	engine.Insert(key)
	logToApp("info", "[T2S] engine.Insert() done")

	started := make(chan struct{})
	go func() {
		close(started)
		logToApp("info", "[T2S] engine.Start() entering...")
		engine.Start()
		logToApp("info", "[T2S] engine.Start() exited")
	}()
	<-started
	time.Sleep(500 * time.Millisecond)
	logToApp("info", "[T2S] engine running")

	t := &tun2Socks{running: true, nfd: nfd}
	return t, nil
}

func (t *tun2Socks) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		logToApp("info", "[T2S] Close() — already stopped")
		return nil
	}
	t.running = false
	logToApp("info", "[T2S] Close() — calling engine.Stop()...")

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logToApp("error", fmt.Sprintf("[T2S] engine.Stop() panicked: %v", r))
			}
		}()
		engine.Stop()
		close(done)
	}()

	select {
	case <-done:
		logToApp("info", "[T2S] engine.Stop() done successfully")
	case <-time.After(2 * time.Second):
		logToApp("error", "[T2S] engine.Stop() timed out!")
	}

	if t.nfd > 0 {
		syscall.Close(t.nfd)
		logToApp("info", fmt.Sprintf("[T2S] syscall.Close(%d) called as fallback", t.nfd))
	}
	logToApp("info", "[T2S] Close() — fully finished")
	return nil
}
