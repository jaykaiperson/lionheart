package golib

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
)

const versionStr = "2.0.0-stub"

type session struct {
	closed chan struct{}
}

func (s *session) OpenStream() (net.Conn, error) {
	return nil, fmt.Errorf("stub: session not implemented")
}

func (s *session) Close() error {
	select {
	case <-s.closed:
	default:
		close(s.closed)
	}
	return nil
}

func newSession(peer, pw string, logger func(string, string)) (*session, error) {
	logger("INFO", fmt.Sprintf("Stub session to %s", peer))
	return &session{closed: make(chan struct{})}, nil
}

func decodeSmartKey(smartKey string) (string, string, error) {
	urlSafe := strings.NewReplacer("-", "+", "_", "/").Replace(smartKey)
	if m := len(urlSafe) % 4; m != 0 {
		urlSafe += strings.Repeat("=", 4-m)
	}
	raw, err := base64.StdEncoding.DecodeString(urlSafe)
	if err != nil {
		return "", "", fmt.Errorf("base64: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid smart key format")
	}
	return parts[0], parts[1], nil
}
