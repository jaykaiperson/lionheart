package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/armon/go-socks5"
	"github.com/hashicorp/yamux"
	"github.com/spf13/cobra"
	"github.com/xtaci/kcp-go/v5"

	"github.com/lionheart-vpn/lionheart/cmd/lionheart/cli"
	"github.com/lionheart-vpn/lionheart/core"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start Lionheart server",
	Long:  "Start the Lionheart private tunnel.",
	RunE:  runServerCmd,
}

var (
	srvPort   string
	srvMTU    int
	srvDNS    string
	srvPing   string
	srvDryRun bool
	srvConf   string
)

func init() {
	serverCmd.Flags().StringVar(&srvPort, "port", "", "Server port (default: from app.json or 8443)")
	serverCmd.Flags().IntVar(&srvMTU, "mtu", 0, "MTU size (default: from app.json or 1500)")
	serverCmd.Flags().StringVar(&srvDNS, "dns", "", "DNS server (default: from app.json or 1.1.1.1)")
	serverCmd.Flags().StringVar(&srvPing, "ping", "", "Ping URL (default: from app.json or https://cp.cloudflare.com)")
	serverCmd.Flags().BoolVar(&srvDryRun, "dry-run", false, "Show config and exit without starting server")
	serverCmd.Flags().StringVar(&srvConf, "conf", "", "Path to json config file")
}

func loadServerConfig(cmd *cobra.Command) (port string, dns string, mtu int, ping string) {
	port = core.DefPort
	dns = "1.1.1.1"
	mtu = 1500
	ping = "https://cp.cloudflare.com"

	appPath := srvConf
	if appPath == "" {
		appPath = core.FindAppConfigPath()
	}
	if appPath != "" {
		if cfg, err := core.LoadAppConfig(appPath); err == nil {
			if cfg.ServerPort != "" {
				port = cfg.ServerPort
			}
			if cfg.DefaultDNS != "" {
				dns = cfg.DefaultDNS
			}
			if cfg.DefaultMTU > 0 {
				mtu = cfg.DefaultMTU
			}
			if cfg.PingURL != "" {
				ping = cfg.PingURL
			}
		}
	}

	if cmd.Flags().Changed("port") {
		port = srvPort
	}
	if cmd.Flags().Changed("mtu") {
		mtu = srvMTU
	}
	if cmd.Flags().Changed("dns") {
		dns = srvDNS
	}
	if cmd.Flags().Changed("ping") {
		ping = srvPing
	}

	return
}

func runServerCmd(cmd *cobra.Command, args []string) error {
	port, dns, mtu, ping := loadServerConfig(cmd)

	fmt.Fprintf(cmd.OutOrStdout(), "Port: %s\n", port)
	fmt.Fprintf(cmd.OutOrStdout(), "DNS:  %s\n", dns)
	fmt.Fprintf(cmd.OutOrStdout(), "MTU:  %d\n", mtu)
	fmt.Fprintf(cmd.OutOrStdout(), "Ping: %s\n", ping)

	if srvDryRun {
		return nil
	}

	addr := "0.0.0.0:" + port

	checkConn, err := net.ListenPacket("udp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: port %s already in use\n", port)
		os.Exit(1)
	}
	checkConn.Close()

	b := make([]byte, 16)
	rand.Read(b)
	pw := hex.EncodeToString(b)

	fmt.Printf("Password: %s\n", pw)

	ip := cli.PubIP()
	if ip != "" {
		cli.PrintSmartKey(ip, port, pw)
	}

	blk, _ := kcp.NewAESBlockCrypt(core.DeriveKey(pw))
	l, err := kcp.ListenWithOptions(addr, blk, 10, 3)
	if err != nil {
		return fmt.Errorf("KCP listen: %w", err)
	}

	cli.InitLogger()
	cli.Inf("Server: %s", addr)

	srv, _ := socks5.New(&socks5.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		fmt.Println()
		cli.Inf("Exiting...")
		cancel()
		go func() { time.Sleep(2 * time.Second); os.Exit(0) }()
		<-sig
		os.Exit(0)
	}()
	go func() { <-ctx.Done(); l.Close() }()

	var wg sync.WaitGroup
	for {
		s, err := l.AcceptKCP()
		if err != nil {
			select {
			case <-ctx.Done():
				wg.Wait()
				return nil
			default:
				continue
			}
		}
		s.SetNoDelay(1, 10, 2, 1)
		s.SetWindowSize(1024, 1024)
		s.SetStreamMode(true)

		wg.Add(1)
		go func(s *kcp.UDPSession) {
			defer wg.Done()
			defer s.Close()
			ym, err := yamux.Server(s, core.YmxCfg())
			if err != nil {
				return
			}
			defer ym.Close()
			cli.Inf("← %s", s.RemoteAddr())
			for {
				st, err := ym.AcceptStream()
				if err != nil {
					cli.Inf("✕ %s", s.RemoteAddr())
					return
				}
				go srv.ServeConn(st)
			}
		}(s)
	}
}
