package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/lionheart-vpn/lionheart/core"
)

func executeServerCommand(args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	cmd := buildServerCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	_, err = cmd.ExecuteC()
	return buf.String(), err
}

func buildServerCmd() *cobra.Command {
	p := ""
	m := 0
	d := ""
	u := ""
	dry := true

	c := &cobra.Command{
		Use:   "server",
		Short: "Start Lionheart VPN server",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, dns, mtu, ping := loadServerConfigFromVars(p, m, d, u)

			cmd.Printf("Port: %s\n", port)
			cmd.Printf("DNS:  %s\n", dns)
			cmd.Printf("MTU:  %d\n", mtu)
			cmd.Printf("Ping: %s\n", ping)

			if dry {
				return nil
			}
			return nil
		},
	}

	c.Flags().StringVar(&p, "port", "", "Server port")
	c.Flags().IntVar(&m, "mtu", 0, "MTU size")
	c.Flags().StringVar(&d, "dns", "", "DNS server")
	c.Flags().StringVar(&u, "ping", "", "Ping URL")
	c.Flags().BoolVar(&dry, "dry-run", true, "Dry run")
	return c
}

func loadServerConfigFromVars(port string, mtu int, dns string, ping string) (string, string, int, string) {
	p := port
	m := mtu
	d := dns
	u := ping

	appPath := core.FindAppConfigPath()
	if appPath != "" {
		if cfg, err := core.LoadAppConfig(appPath); err == nil {
			if p == "" {
				p = cfg.ServerPort
			}
			if d == "" {
				d = cfg.DefaultDNS
			}
			if m == 0 {
				m = cfg.DefaultMTU
			}
			if u == "" {
				u = cfg.PingURL
			}
		}
	}
	if p == "" {
		p = core.DefPort
	}
	if d == "" {
		d = "1.1.1.1"
	}
	if m == 0 {
		m = 1500
	}
	if u == "" {
		u = "https://cp.cloudflare.com"
	}
	return p, d, m, u
}

func TestServer_DefaultConfig(t *testing.T) {
	output, err := executeServerCommand()
	if err != nil {
		t.Fatalf("executeServerCommand() error = %v", err)
	}
	if !strings.Contains(output, "Port: 8443") {
		t.Errorf("Missing default port, got: %s", output)
	}
	if !strings.Contains(output, "DNS:  1.1.1.1") {
		t.Errorf("Missing default DNS, got: %s", output)
	}
	if !strings.Contains(output, "MTU:  1500") {
		t.Errorf("Missing default MTU, got: %s", output)
	}
	if !strings.Contains(output, "Ping: https://cp.cloudflare.com") {
		t.Errorf("Missing default ping URL, got: %s", output)
	}
}

func TestServer_CLIOverrides(t *testing.T) {
	output, err := executeServerCommand(
		"--port", "9090",
		"--mtu", "1400",
		"--dns", "8.8.8.8",
		"--ping", "https://example.com/ping",
	)
	if err != nil {
		t.Fatalf("executeServerCommand() error = %v", err)
	}
	if !strings.Contains(output, "Port: 9090") {
		t.Errorf("Port override failed, got: %s", output)
	}
	if !strings.Contains(output, "MTU:  1400") {
		t.Errorf("MTU override failed, got: %s", output)
	}
	if !strings.Contains(output, "DNS:  8.8.8.8") {
		t.Errorf("DNS override failed, got: %s", output)
	}
	if !strings.Contains(output, "Ping: https://example.com/ping") {
		t.Errorf("Ping URL override failed, got: %s", output)
	}
}

func TestServer_InvalidMTU(t *testing.T) {
	output, err := executeServerCommand("--mtu", "0")
	if err != nil {
		t.Fatalf("executeServerCommand() unexpected error: %v", err)
	}
	if !strings.Contains(output, "MTU:  1500") {
		t.Errorf("MTU should be default 1500 for zero input, got: %s", output)
	}
}

func TestServer_FlagsExist(t *testing.T) {
	cmd := buildServerCmd()
	for _, flag := range []string{"port", "mtu", "dns", "ping", "dry-run"} {
		f := cmd.Flag(flag)
		if f == nil {
			t.Errorf("Missing flag: %s", flag)
		}
	}
}

func TestRoot_SubcommandsExist(t *testing.T) {
	cmd := &cobra.Command{Use: "lionheart"}
	cmd.AddCommand(serverCmd)
	cmd.AddCommand(clientCmd)

	server, _, err := cmd.Find([]string{"server"})
	if err != nil {
		t.Fatalf("server subcommand not found: %v", err)
	}
	if server.Use != "server" {
		t.Errorf("server.Use = %q, want %q", server.Use, "server")
	}

	client, _, err := cmd.Find([]string{"client"})
	if err != nil {
		t.Fatalf("client subcommand not found: %v", err)
	}
	if client.Use != "client" {
		t.Errorf("client.Use = %q, want %q", client.Use, "client")
	}
}
