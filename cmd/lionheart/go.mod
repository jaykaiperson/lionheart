module github.com/lionheart-vpn/lionheart/cmd/lionheart

go 1.22

require (
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/hashicorp/yamux v0.1.2
	github.com/lionheart-vpn/lionheart/core v0.0.0
	github.com/xtaci/kcp-go/v5 v5.6.71
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/klauspost/reedsolomon v1.12.0 // indirect
	github.com/pion/dtls/v3 v3.0.7 // indirect
	github.com/pion/logging v0.2.4 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/stun/v3 v3.0.1 // indirect
	github.com/pion/transport/v3 v3.0.8 // indirect
	github.com/pion/transport/v4 v4.0.1 // indirect
	github.com/pion/turn/v4 v4.1.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sagernet/sing v0.6.0 // indirect
	github.com/sagernet/sing-box v1.11.0 // indirect
	github.com/sagernet/sing-dns v0.4.0 // indirect
	github.com/sagernet/sing-tun v0.6.0 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/time v0.14.0 // indirect
)

replace github.com/lionheart-vpn/lionheart/core => ../../core
