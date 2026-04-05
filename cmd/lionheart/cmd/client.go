package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/lionheart-vpn/lionheart/cmd/lionheart/cli"
	"github.com/lionheart-vpn/lionheart/core"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start Lionheart client",
	Long:  "Connect to a Lionheart tunnel using a smart key or config file.",
	RunE:  runClientCmd,
}

var (
	clSmartKey  string
	clSocksPort string
)

func init() {
	clientCmd.Flags().StringVar(&clSmartKey, "key", "", "Smart key from server (overrides config file)")
	clientCmd.Flags().StringVar(&clSocksPort, "port", "1080", "Local SOCKS5 port (default: 1080)")
}

func runClientCmd(cmd *cobra.Command, args []string) error {
	var peer, pw string

	if clSmartKey != "" {
		var err error
		peer, pw, err = core.ParseSmartKey(clSmartKey)
		if err != nil {
			return fmt.Errorf("invalid smart key: %w", err)
		}
	} else {
		cfg := cli.LoadCfg()
		if cfg == nil || cfg.Role != "client" {
			return fmt.Errorf("no client config found. Run 'lionheart client --key <smart-key>' or run server first")
		}
		peer = cfg.ClientPeer
		pw = cfg.Password
	}

	cli.InitLogger()
	cli.KillSiblings()

	cache := &core.CredsCache{}
	sess := &core.Session{}
	rch := make(chan struct{}, 1)

	ym, cl, err := core.Establish(cache, peer, pw, true)
	if err != nil {
		return fmt.Errorf("tunnel: %w", err)
	}
	sess.Set(ym, cl)

	go core.HealthLoop(context.Background(), sess, rch)
	go core.ReconnectLoop(context.Background(), sess, cache, peer, pw, rch)

	l, err := net.Listen("tcp", "0.0.0.0:"+clSocksPort)
	if err != nil {
		return fmt.Errorf("port %s: %w", clSocksPort, err)
	}

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

	fmt.Println()
	cli.Inf("   Tunnel active!")
	cli.Inf("   Local:    \033[32m127.0.0.1:%s\033[0m", clSocksPort)
	cli.Inf("   LAN:      \033[32m%s:%s\033[0m", cli.LocalIP(), clSocksPort)
	cli.Inf("   Ctrl+C — exit")
	fmt.Println()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				sess.Stop()
				return nil
			default:
				continue
			}
		}
		go func(c net.Conn) {
			defer c.Close()
			y, ok := sess.Get()
			if !ok || y == nil {
				return
			}
			s, err := y.OpenStream()
			if err != nil {
				sess.Down()
				select {
				case rch <- struct{}{}:
				default:
				}
				return
			}
			defer s.Close()
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); io.Copy(s, c) }()
			go func() { defer wg.Done(); io.Copy(c, s) }()
			wg.Wait()
		}(conn)
	}
}
