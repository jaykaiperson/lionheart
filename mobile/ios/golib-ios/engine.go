package golib

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

type vpnEngine struct {
	ctx       context.Context
	cancel    context.CancelFunc
	peer      string
	pw        string
	tunFd     int
	mtu       int
	dns       string
	status    string
	sess      *session
	connCount atomic.Int64
	txBytes   atomic.Int64
	rxBytes   atomic.Int64
}

func (e *vpnEngine) run() error {
	e.status = "connecting"
	emit("connecting")
	sess, err := newSession(e.peer, e.pw, logToApp)
	if err != nil {
		e.status = "error"
		emit("error")
		return fmt.Errorf("session: %w", err)
	}
	e.sess = sess
	ln, err := net.Listen("tcp", "127.0.0.1:1080")
	if err != nil {
		return fmt.Errorf("socks listen: %w", err)
	}
	defer ln.Close()
	e.status = "connected"
	emit("connected")
	logToApp("INFO", "SOCKS5 ready on 127.0.0.1:1080")
	var wg sync.WaitGroup
	go e.serveSocks5(ln)
	<-e.ctx.Done()
	ln.Close()
	wg.Wait()
	return nil
}

func (e *vpnEngine) stop() {
	if e.cancel != nil { e.cancel() }
	if e.sess != nil { e.sess.Close() }
	e.status = "disconnected"
	emit("disconnected")
}

func (e *vpnEngine) serveSocks5(ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil { return }
		e.connCount.Add(1)
		go e.handleSocks5(c)
	}
}

func (e *vpnEngine) handleSocks5(c net.Conn) {
	defer c.Close()
	defer e.connCount.Add(-1)
	buf := make([]byte, 512)
	n, err := c.Read(buf)
	if err != nil || n < 3 { return }
	c.Write([]byte{0x05, 0x00})
	n, err = c.Read(buf)
	if err != nil || n < 7 { return }
	if buf[1] == 0x01 { e.handleConnect(c, buf[:n]) }
}

func (e *vpnEngine) handleConnect(c net.Conn, req []byte) {
	stream, err := e.sess.OpenStream()
	if err != nil {
		c.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer stream.Close()
	stream.Write(req)
	reply := make([]byte, 256)
	n, err := stream.Read(reply)
	if err != nil { return }
	c.Write(reply[:n])
	done := make(chan struct{}, 2)
	go func() { written, _ := io.Copy(stream, c); e.txBytes.Add(written); done <- struct{}{} }()
	go func() { written, _ := io.Copy(c, stream); e.rxBytes.Add(written); done <- struct{}{} }()
	select {
	case <-done:
	case <-e.ctx.Done():
	}
}
