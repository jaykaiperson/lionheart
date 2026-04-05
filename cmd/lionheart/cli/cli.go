package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lionheart-vpn/lionheart/core"
)

const cfgFile = "config.json"

var banner = "\033[38;5;208m" + `
  ▄▄▄                                               
 ▀██▀                    █▄                     █▄  
  ██      ▀▀       ▄     ██                ▄    ▄██▄
  ██      ██ ▄███▄ ████▄ ████▄ ▄█▀█▄ ▄▀▀█▄ ████▄ ██ 
  ██      ██ ██ ██ ██ ██ ██ ██ ██▄█▀ ▄█▀██ ██    ██ 
 ████████▄██▄▀███▀▄██ ▀█▄██ ██▄▀█▄▄▄▄▀█▄██▄█▀   ▄██ 
` + "\033[0m                                              v" + core.Version + "\n"

type Cfg struct {
	Role, Password, ServerListen, ClientPeer string
}

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

func InitLogger() {
	core.SetLogger(cliLogger{})
}

func Inf(f string, a ...any) { core.Log.Info(fmt.Sprintf(f, a...)) }
func Wrn(f string, a ...any) { core.Log.Warn(fmt.Sprintf(f, a...)) }
func Die(f string, a ...any) {
	core.Log.Error(fmt.Sprintf(f, a...))
	os.Exit(1)
}

type Spin struct {
	msg string
	ch  chan struct{}
}

func Spinner(msg string) *Spin {
	s := &Spin{msg, make(chan struct{})}
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

func (s *Spin) Done() { close(s.ch); time.Sleep(40 * time.Millisecond) }

func Ask(p string) string {
	fmt.Print(p)
	v, _ := io.ReadAll(io.LimitReader(os.Stdin, 1024))
	return strings.TrimSpace(string(v))
}

func PubIP() string {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, u := range []string{"https://api.ipify.org", "https://ifconfig.me/ip"} {
		resp, err := client.Get(u)
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		ip := strings.TrimSpace(string(b))
		if strings.Contains(ip, ".") || strings.Contains(ip, ":") {
			return ip
		}
	}
	return ""
}

func LocalIP() string {
	aa, _ := net.InterfaceAddrs()
	for _, a := range aa {
		if ip, ok := a.(*net.IPNet); ok && !ip.IP.IsLoopback() && ip.IP.To4() != nil {
			return ip.IP.String()
		}
	}
	return "?"
}

func SaveCfg(c *Cfg) {
	d, _ := json.MarshalIndent(c, "", "  ")
	tmp := cfgFile + ".tmp"
	os.WriteFile(tmp, d, 0644)
	os.Rename(tmp, cfgFile)
}

func LoadCfg() *Cfg {
	if _, e := os.Stat(cfgFile); os.IsNotExist(e) {
		return nil
	}
	d, _ := os.ReadFile(cfgFile)
	var c Cfg
	json.Unmarshal(d, &c)
	return &c
}

func PrintQR(data string) {
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

func PrintSmartKey(host, port, pw string) {
	smartKey := core.EncodeSmartKey(host, port, pw)
	fmt.Printf("\n\033[33m┌─── SMART KEY ───────────────────────────────────┐\033[0m\n")
	fmt.Printf("\033[33m│\033[0m \033[32m%s\033[0m\n", smartKey)
	fmt.Printf("\033[33m└─────────────────────────────────────────────────┘\033[0m\n\n")
	fmt.Println("\033[33m┌─── QR CODE ─────────────────────────────────────┐\033[0m")
	PrintQR(smartKey)
	fmt.Println("\033[33m└─────────────────────────────────────────────────┘\033[0m")
	fmt.Println()
}

func SelfExe() string { p, _ := os.Executable(); a, _ := filepath.Abs(p); return a }
func IsSystemd() bool { return os.Getenv("INVOCATION_ID") != "" }

func KillSiblings() {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return
	}
	if !IsSystemd() && runtime.GOOS == "linux" {
		exec.Command("systemctl", "stop", "lionheart.service").Run()
	}
	myPid := os.Getpid()
	myExe := filepath.Base(SelfExe())
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
			Inf("Killed PID %d", pid)
		}
	}
	time.Sleep(300 * time.Millisecond)
}

func ReplaceService() {
	if runtime.GOOS != "linux" || IsSystemd() {
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
	exe := SelfExe()
	if strings.Contains(string(data), exe) {
		return
	}
	InstallService(exe, filepath.Dir(exe))
	Inf("Service updated → %s", filepath.Base(exe))
}

func InstallService(exe, workDir string) {
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
		Wrn("Cannot create service: %v", err)
		return
	}
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "lionheart.service").Run()
}

func StartService() {
	exec.Command("systemctl", "restart", "lionheart.service").Run()
}

func PrintBanner() {
	if runtime.GOOS == "windows" {
		fmt.Println("Lionheart VPN v" + core.Version)
	} else {
		fmt.Print(banner)
	}
}
