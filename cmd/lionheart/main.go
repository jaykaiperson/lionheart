// lionheart v1.2 — CLI client/server
// Imports shared logic from core/ package.
package main

import (
    "bufio"
    "context"
    "crypto/rand"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "net/http"
    "os"
    "os/exec"
    "os/signal"
    "path/filepath"
    "runtime"
    "strconv"
    "strings"
    "sync"
    "syscall"
    "time"

    "github.com/armon/go-socks5"
    "github.com/hashicorp/yamux"
    "github.com/xtaci/kcp-go/v5"

    "github.com/lionheart-vpn/lionheart/core"
)

const (
    cfgFile = "config.json"
    banner  = "\033[38;5;208m" + `
  ▄▄▄                                               
 ▀██▀                    █▄                     █▄  
  ██      ▀▀       ▄     ██                ▄    ▄██▄
  ██      ██ ▄███▄ ████▄ ████▄ ▄█▀█▄ ▄▀▀█▄ ████▄ ██ 
  ██      ██ ██ ██ ██ ██ ██ ██ ██▄█▀ ▄█▀██ ██    ██ 
 ████████▄██▄▀███▀▄██ ▀█▄██ ██▄▀█▄▄▄▄▀█▄██▄█▀   ▄██ 
` + "\033[0m                                              v" + core.Version + "\n"
)

type Cfg struct {
    Role, Password, ServerListen, ClientPeer string
}

// ─── CLI Logger (implements core.Logger) ───

type cliLogger struct{}

var logMu sync.Mutex

func out(pre, color, msg string) {
    logMu.Lock()
    defer logMu.Unlock()
    fmt.Printf("\r\033[K[%s] \033[%sm%s\033[0m %s\n", time.Now().Format("15:04:05"), color, pre, msg)
}

func (cliLogger) Info(msg string)  { out("INFO", "36", msg) }
func (cliLogger) Warn(msg string)  { out("WARN", "33", msg) }
func (cliLogger) Error(msg string) { out("FAIL", "31", msg) }

func inf(f string, a ...any) { core.Log.Info(fmt.Sprintf(f, a...)) }
func wrn(f string, a ...any) { core.Log.Warn(fmt.Sprintf(f, a...)) }
func die(f string, a ...any) {
    core.Log.Error(fmt.Sprintf(f, a...))
    os.Exit(1)
}

// ─── Spinner ───

type spin struct {
    msg string
    ch  chan struct{}
}

func spinner(msg string) *spin {
    s := &spin{msg, make(chan struct{})}
    frames := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
    t0 := time.Now()
    go func() {
        i := 0
        for {
            select {
            case <-s.ch:
                logMu.Lock()
                fmt.Printf("\r\033[K\033[32m[ ✓ ]\033[0m %s \033[90m%ds\033[0m\n", s.msg, int(time.Since(t0).Seconds()))
                logMu.Unlock()
                return
            case <-time.After(80 * time.Millisecond):
                r := []rune(frames)
                logMu.Lock()
                fmt.Printf("\r\033[K\033[36m[%c]\033[0m %s \033[90m%ds\033[0m", r[i%len(r)], s.msg, int(time.Since(t0).Seconds()))
                logMu.Unlock()
                i++
            }
        }
    }()
    return s
}
func (s *spin) done() { close(s.ch); time.Sleep(40 * time.Millisecond) }

func ask(p string) string {
    fmt.Print(p)
    v, _ := bufio.NewReader(os.Stdin).ReadString('\n')
    return strings.TrimSpace(v)
}

// ─── Helpers ───

// ИСПРАВЛЕНО: Теперь использует стандартный HTTP с резервным сервером для обхода брандмауэров Windows
func pubIP() string {
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get("https://api.ipify.org")
    if err != nil {
        resp, err = client.Get("https://ifconfig.me/ip")
    }
    if err == nil {
        defer resp.Body.Close()
        b, _ := io.ReadAll(resp.Body)
        ip := strings.TrimSpace(string(b))
        if strings.Contains(ip, ".") || strings.Contains(ip, ":") {
            return ip
        }
    }
    return ""
}

func localIP() string {
    aa, _ := net.InterfaceAddrs()
    for _, a := range aa {
        if ip, ok := a.(*net.IPNet); ok && !ip.IP.IsLoopback() && ip.IP.To4() != nil {
            return ip.IP.String()
        }
    }
    return "?"
}

func saveCfg(c *Cfg) {
    d, _ := json.MarshalIndent(c, "", "  ")
    tmp := cfgFile + ".tmp"
    os.WriteFile(tmp, d, 0644)
    os.Rename(tmp, cfgFile)
}

func loadCfg() *Cfg {
    if _, e := os.Stat(cfgFile); os.IsNotExist(e) {
        return wizard()
    }
    d, _ := os.ReadFile(cfgFile)
    var c Cfg
    json.Unmarshal(d, &c)
    return &c
}

// ─── QR Code generation (terminal) ───

func printQR(data string) {
    path, err := exec.LookPath("qrencode")
    if err == nil && path != "" {
        cmd := exec.Command("qrencode", "-t", "UTF8", "-o", "-", data)
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        if cmd.Run() == nil {
            return
        }
    }
    fmt.Println("\033[90m(Установите qrencode для QR-кода: apt install qrencode / brew install qrencode)\033[0m")
}

// ─── Self-management ───

func selfExe() string { p, _ := os.Executable(); a, _ := filepath.Abs(p); return a }
func isSystemd() bool { return os.Getenv("INVOCATION_ID") != "" }

func killSiblings() {
    if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
        return
    }
    if !isSystemd() && runtime.GOOS == "linux" {
        exec.Command("systemctl", "stop", "lionheart.service").Run()
    }
    myPid := os.Getpid()
    myExe := filepath.Base(selfExe())
    out, err := exec.Command("pgrep", "-f", myExe).Output()
    if err != nil {
        return
    }
    for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
        pid, err := strconv.Atoi(strings.TrimSpace(line))
        if err != nil || pid == myPid {
            continue
        }
        if p, err := os.FindProcess(pid); err == nil {
            p.Kill()
            inf("Killed PID %d", pid)
        }
    }
    time.Sleep(300 * time.Millisecond)
}

func replaceService() {
    if runtime.GOOS != "linux" || isSystemd() {
        return
    }
    svcPath := "/etc/systemd/system/lionheart.service"
    if _, err := os.Stat(svcPath); os.IsNotExist(err) {
        return
    }
    data, err := os.ReadFile(svcPath)
    if err != nil {
        return
    }
    exe := selfExe()
    if strings.Contains(string(data), exe) {
        return
    }
    installService(exe, filepath.Dir(exe))
    inf("Service updated → %s", filepath.Base(exe))
}

func installService(exe, workDir string) {
    unit := fmt.Sprintf(`[Unit]
Description=Lionheart v%s
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=%s
ExecStart=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target`, core.Version, workDir, exe)

    if err := os.WriteFile("/etc/systemd/system/lionheart.service", []byte(unit), 0644); err != nil {
        wrn("Cannot create service: %v", err)
        return
    }
    exec.Command("systemctl", "daemon-reload").Run()
    exec.Command("systemctl", "enable", "lionheart.service").Run()
}

func startService() {
    exec.Command("systemctl", "restart", "lionheart.service").Run()
}

// ─── Setup Wizard ───

func wizard() *Cfg {
    c := &Cfg{}
    if ask("Mode (1-server, 2-client): ") == "1" {
        c.Role, c.ServerListen = "server", "0.0.0.0:"+core.DefPort
        b := make([]byte, 16)
        rand.Read(b)
        c.Password = hex.EncodeToString(b)
        sp := spinner("Detecting public IP")
        ip := pubIP()
        sp.done()
        if ip == "" || !strings.Contains(ip, ".") {
            ip = ask("IP manually: ")
        }
        smartKey := core.EncodeSmartKey(ip, core.DefPort, c.Password)

        fmt.Printf("\n\033[33m┌─── SMART KEY ───────────────────────────────────┐\033[0m\n")
        fmt.Printf("\033[33m│\033[0m \033[32m%s\033[0m\n", smartKey)
        fmt.Printf("\033[33m└─────────────────────────────────────────────────┘\033[0m\n\n")

        fmt.Println("\033[33m┌─── QR CODE ─────────────────────────────────────┐\033[0m")
        printQR(smartKey)
        fmt.Println("\033[33m└─────────────────────────────────────────────────┘\033[0m")
        fmt.Println()

        saveCfg(c)
        if runtime.GOOS == "linux" && ask("Install as service? (y/n): ") == "y" {
            installService(selfExe(), filepath.Dir(selfExe()))
            startService()
            fmt.Println("Service started!")
            os.Exit(0)
        }
        ask("Enter to start...")
    } else {
        c.Role = "client"
        d, err := base64.RawURLEncoding.DecodeString(ask("Smart-key: "))
        if err != nil {
            die("Invalid key")
        }
        p := strings.SplitN(string(d), "|", 2)
        if len(p) != 2 {
            die("Corrupted key")
        }
        c.ClientPeer = p[0]
        if !strings.Contains(c.ClientPeer, ":") {
            c.ClientPeer += ":" + core.DefPort
        }
        c.Password = p[1]
        saveCfg(c)
    }
    return c
}

// ─── Client ───

func runClient(ctx context.Context, peer, pw string) {
    killSiblings()

    cache := &core.CredsCache{}
    sess := &core.Session{}
    rch := make(chan struct{}, 1)

    ym, cl, err := core.Establish(cache, peer, pw, true)
    if err != nil {
        die("Tunnel: %v", err)
    }
    sess.Set(ym, cl)

    go core.HealthLoop(ctx, sess, rch)
    go core.ReconnectLoop(ctx, sess, cache, peer, pw, rch)

    l, err := net.Listen("tcp", "0.0.0.0:1080")
    if err != nil {
        die("Port 1080: %v", err)
    }
    go func() { <-ctx.Done(); l.Close() }()

    fmt.Println()
    inf("   Tunnel active!")
    inf("   Local:    \033[32m127.0.0.1:1080\033[0m")
    inf("   LAN:      \033[32m%s:1080\033[0m", localIP())
    inf("   Ctrl+C — exit")
    fmt.Println()

    for {
        conn, err := l.Accept()
        if err != nil {
            select {
            case <-ctx.Done():
                sess.Stop()
                return
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

// ─── Server ───

func runServer(ctx context.Context, addr, pw string) {
    killSiblings()

    blk, _ := kcp.NewAESBlockCrypt(core.DeriveKey(pw))
    l, err := kcp.ListenWithOptions(addr, blk, 10, 3)
    if err != nil {
        die("KCP: %v", err)
    }

    inf("Server: \033[32m%s\033[0m", addr)

    host, port, _ := net.SplitHostPort(addr)
    if host == "0.0.0.0" || host == "" {
        sp := spinner("Detecting public IP")
        host = pubIP()
        sp.done()
    }
    if host != "" {
        smartKey := core.EncodeSmartKey(host, port, pw)
        fmt.Printf("\n\033[33m┌─── SMART KEY ───────────────────────────────────┐\033[0m\n")
        fmt.Printf("\033[33m│\033[0m \033[32m%s\033[0m\n", smartKey)
        fmt.Printf("\033[33m└─────────────────────────────────────────────────┘\033[0m\n\n")
        printQR(smartKey)
        fmt.Println()
    }

    srv, _ := socks5.New(&socks5.Config{})
    go func() { <-ctx.Done(); l.Close() }()

    var wg sync.WaitGroup
    for {
        s, err := l.AcceptKCP()
        if err != nil {
            select {
            case <-ctx.Done():
                wg.Wait()
                return
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
            inf("← \033[33m%s\033[0m", s.RemoteAddr())
            for {
                st, err := ym.AcceptStream()
                if err != nil {
                    inf("✕ \033[33m%s\033[0m", s.RemoteAddr())
                    return
                }
                go srv.ServeConn(st)
            }
        }(s)
    }
}

// ─── Main ───

func main() {
    if runtime.GOOS == "windows" {
        fmt.Println("Lionheart VPN v" + core.Version)
    } else {
        fmt.Print(banner)
    }

    core.SetLogger(cliLogger{})

    ctx, cancel := context.WithCancel(context.Background())
    sig := make(chan os.Signal, 2)
    signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
    go func() {
        <-sig
        fmt.Println()
        inf("Exiting...")
        cancel()
        go func() { time.Sleep(2 * time.Second); os.Exit(0) }()
        <-sig
        os.Exit(0)
    }()

    cfg := loadCfg()

    if runtime.GOOS == "linux" {
        replaceService()
    }

    if cfg.Role == "server" {
        runServer(ctx, cfg.ServerListen, cfg.Password)
    } else {
        runClient(ctx, cfg.ClientPeer, cfg.Password)
    }
}