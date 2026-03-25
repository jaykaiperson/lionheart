// lionheart v1.3 — SOCKS5 over KCP via WB Stream TURN (no browser)
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"github.com/pion/turn/v4"
	"github.com/xtaci/kcp-go/v5"
)

const (
	V           = "1.3"
	cfgFile     = "config.json"
	defPort     = "8443"
	wbBase      = "https://stream.wb.ru"
	wbUA        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	maxBackoff  = 60 * time.Second
	healthEvery = 15 * time.Second
	credsTTL    = 5 * time.Minute
	banner      = "\033[38;5;208m" + `
  ▄▄▄                                               
 ▀██▀                    █▄                     █▄  
  ██      ▀▀       ▄     ██                ▄    ▄██▄
  ██      ██ ▄███▄ ████▄ ████▄ ▄█▀█▄ ▄▀▀█▄ ████▄ ██ 
  ██      ██ ██ ██ ██ ██ ██ ██ ██▄█▀ ▄█▀██ ██    ██ 
 ████████▄██▄▀███▀▄██ ▀█▄██ ██▄▀█▄▄▄▄▀█▄██▄█▀   ▄██ 
` + "\033[0m                                              v" + V + "\n"
)

type Cfg struct {
	Role, Password, ServerListen, ClientPeer string
}
type turnCred struct{ URL, User, Pass string }

// ─── UI ───

var mu sync.Mutex

func out(pre, color, msg string) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Printf("\r\033[K[%s] \033[%sm%s\033[0m %s\n", time.Now().Format("15:04:05"), color, pre, msg)
}
func inf(f string, a ...any) { out("INFO", "36", fmt.Sprintf(f, a...)) }
func wrn(f string, a ...any) { out("WARN", "33", fmt.Sprintf(f, a...)) }
func die(f string, a ...any) {
	out("FAIL", "31", fmt.Sprintf(f, a...))
	os.Exit(1)
}

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
				mu.Lock()
				fmt.Printf("\r\033[K\033[32m[ ✓ ]\033[0m %s \033[90m%ds\033[0m\n", s.msg, int(time.Since(t0).Seconds()))
				mu.Unlock()
				return
			case <-time.After(80 * time.Millisecond):
				r := []rune(frames)
				mu.Lock()
				fmt.Printf("\r\033[K\033[36m[%c]\033[0m %s \033[90m%ds\033[0m", r[i%len(r)], s.msg, int(time.Since(t0).Seconds()))
				mu.Unlock()
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

func pubIP() string {
	r, e := (&http.Client{Timeout: 8 * time.Second}).Get("https://api.ipify.org")
	if e == nil && r != nil {
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return strings.TrimSpace(string(b))
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

func key(pw string) []byte { h := sha256.Sum256([]byte(pw)); return h[:] }

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

// ─── Self-management ───

func selfExe() string { p, _ := os.Executable(); a, _ := filepath.Abs(p); return a }

// isSystemd returns true if we are running as a systemd service
func isSystemd() bool { return os.Getenv("INVOCATION_ID") != "" }

// killSiblings stops the systemd service (if any) and kills other lionheart processes
func killSiblings() {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return
	}
	// Stop systemd service first (if we're NOT the service ourselves)
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
			inf("Завершён процесс PID %d", pid)
		}
	}
	time.Sleep(300 * time.Millisecond)
}

// replaceService updates the systemd unit file to point to the current binary.
// Only runs when launched manually (not as systemd service itself).
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
		return // already up to date, no-op (service was stopped by killSiblings)
	}
	installService(exe, filepath.Dir(exe))
	inf("Служба обновлена → %s", filepath.Base(exe))
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
WantedBy=multi-user.target`, V, workDir, exe)

	if err := os.WriteFile("/etc/systemd/system/lionheart.service", []byte(unit), 0644); err != nil {
		wrn("Не удалось создать службу: %v", err)
		return
	}
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "lionheart.service").Run()
}

func startService() {
	exec.Command("systemctl", "restart", "lionheart.service").Run()
}

// ─── Setup ───

func wizard() *Cfg {
	c := &Cfg{}
	if ask("Режим (1-сервер, 2-клиент): ") == "1" {
		c.Role, c.ServerListen = "server", "0.0.0.0:"+defPort
		b := make([]byte, 16)
		rand.Read(b)
		c.Password = hex.EncodeToString(b)
		sp := spinner("Определение IP")
		ip := pubIP()
		sp.done()
		if ip == "" || !strings.Contains(ip, ".") {
			ip = ask("IP вручную: ")
		}
		k := base64.RawURLEncoding.EncodeToString([]byte(ip + "|" + c.Password))
		fmt.Printf("\n\033[33m--- СМАРТ-КЛЮЧ ---\033[0m\n\033[32m%s\033[0m\n\033[33m------------------\033[0m\n\n", k)
		saveCfg(c)
		if runtime.GOOS == "linux" && ask("Установить как службу? (y/n): ") == "y" {
			installService(selfExe(), filepath.Dir(selfExe()))
			startService()
			fmt.Println("Служба запущена!")
			os.Exit(0)
		}
		ask("Enter для запуска...")
	} else {
		c.Role = "client"
		d, err := base64.RawURLEncoding.DecodeString(ask("Смарт-ключ: "))
		if err != nil {
			die("Неверный ключ")
		}
		p := strings.SplitN(string(d), "|", 2)
		if len(p) != 2 {
			die("Повреждённый ключ")
		}
		c.ClientPeer = p[0]
		if !strings.Contains(c.ClientPeer, ":") {
			c.ClientPeer += ":" + defPort
		}
		c.Password = p[1]
		saveCfg(c)
	}
	return c
}

// ─── WB Stream ───

type credsCache struct {
	sync.Mutex
	creds []turnCred
	at    time.Time
}

func (c *credsCache) get(force bool) ([]turnCred, error) {
	c.Lock()
	defer c.Unlock()
	if !force && len(c.creds) > 0 && time.Since(c.at) < credsTTL {
		return c.creds, nil
	}
	cr, err := fetchCreds()
	if err != nil {
		return nil, err
	}
	c.creds, c.at = cr, time.Now()
	return cr, nil
}

func wb(cl *http.Client, method, ep string, body []byte, tok string) ([]byte, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	rq, _ := http.NewRequest(method, wbBase+ep, rd)
	rq.Header.Set("User-Agent", wbUA)
	rq.Header.Set("Accept", "application/json")
	rq.Header.Set("Accept-Language", "en-US,en;q=0.9")
	rq.Header.Set("Origin", wbBase)
	rq.Header.Set("Referer", wbBase+"/")
	if body != nil {
		rq.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		rq.Header.Set("Authorization", "Bearer "+tok)
	}
	rs, err := cl.Do(rq)
	if err != nil {
		return nil, err
	}
	defer rs.Body.Close()
	var r io.Reader = rs.Body
	if rs.Header.Get("Content-Encoding") == "gzip" {
		if g, e := gzip.NewReader(rs.Body); e == nil {
			defer g.Close()
			r = g
		}
	}
	b, _ := io.ReadAll(r)
	if rs.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", rs.StatusCode, b)
	}
	return b, nil
}

func fetchCreds() ([]turnCred, error) {
	cl := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{}}}
	nm := fmt.Sprintf("lh_%d", time.Now().UnixMilli()%100000)

	sp := spinner("Подключение к WB Stream")

	// 1. guest register
	rr, err := wb(cl, "POST", "/auth/api/v1/auth/user/guest-register", []byte(`{"displayName":"`+nm+`"}`), "")
	if err != nil {
		sp.done()
		return nil, err
	}
	var reg struct {
		AccessToken string `json:"accessToken"`
	}
	json.Unmarshal(rr, &reg)
	if reg.AccessToken == "" {
		sp.done()
		return nil, fmt.Errorf("нет токена")
	}

	// 2. create room
	rr, err = wb(cl, "POST", "/api-room/api/v2/room",
		[]byte(`{"roomType":"ROOM_TYPE_ALL_ON_SCREEN","roomPrivacy":"ROOM_PRIVACY_FREE"}`), reg.AccessToken)
	if err != nil {
		sp.done()
		return nil, err
	}
	var room struct {
		RoomID string `json:"roomId"`
	}
	json.Unmarshal(rr, &room)

	// 3. join
	wb(cl, "POST", fmt.Sprintf("/api-room/api/v1/room/%s/join", room.RoomID), []byte("{}"), reg.AccessToken)

	// 4. token
	rr, err = wb(cl, "GET", fmt.Sprintf("/api-room-manager/api/v1/room/%s/token?deviceType=PARTICIPANT_DEVICE_TYPE_WEB_DESKTOP&displayName=%s",
		room.RoomID, url.QueryEscape(nm)), nil, reg.AccessToken)
	sp.done()
	if err != nil {
		return nil, err
	}
	var tok struct {
		RoomToken string `json:"roomToken"`
	}
	json.Unmarshal(rr, &tok)

	// 5. LiveKit ICE
	sp = spinner("Согласование ICE (LiveKit)")
	creds, err := lkICE(tok.RoomToken)
	sp.done()
	if err != nil {
		return nil, err
	}
	for _, c := range creds {
		inf("  → %s", c.URL)
	}
	return creds, nil
}

func lkICE(token string) ([]turnCred, error) {
	u := "wss://wbstream01-el.wb.ru:7880/rtc?access_token=" + url.QueryEscape(token) +
		"&auto_subscribe=1&sdk=js&version=2.15.3&protocol=16&adaptive_stream=1"
	conn, _, err := (&websocket.Dialer{
		TLSClientConfig: &tls.Config{}, HandshakeTimeout: 10 * time.Second,
	}).Dial(u, http.Header{"User-Agent": {wbUA}, "Origin": {wbBase}})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	for i := 0; i < 15; i++ {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if c := pbICE(msg); len(c) > 0 {
			return dedup(c), nil
		}
	}
	return nil, fmt.Errorf("TURN не найдены")
}

// ─── Protobuf (minimal) ───

func pbVar(d []byte, o int) (uint64, int) {
	var v uint64
	for s := 0; o < len(d) && s < 64; s += 7 {
		b := d[o]
		o++
		v |= uint64(b&0x7f) << s
		if b < 0x80 {
			return v, o
		}
	}
	return 0, o
}

func pbAll(d []byte, f uint64) (r [][]byte) {
	for o := 0; o < len(d); {
		t, n := pbVar(d, o)
		if n == o {
			break
		}
		o = n
		switch t & 7 {
		case 0:
			_, o = pbVar(d, o)
		case 2:
			l, n := pbVar(d, o)
			o = n
			e := o + int(l)
			if e > len(d) || e < o {
				return
			}
			if t>>3 == f {
				r = append(r, d[o:e])
			}
			o = e
		case 1:
			o += 8
		case 5:
			o += 4
		default:
			return
		}
	}
	return
}

func pbStr(d []byte, f uint64) string {
	if a := pbAll(d, f); len(a) > 0 {
		return string(a[0])
	}
	return ""
}

func pbICE(d []byte) (res []turnCred) {
	for o := 0; o < len(d); {
		t, n := pbVar(d, o)
		if n == o {
			break
		}
		o = n
		switch t & 7 {
		case 0:
			_, o = pbVar(d, o)
		case 2:
			l, n := pbVar(d, o)
			o = n
			e := o + int(l)
			if e > len(d) || e < o {
				return
			}
			inner := d[o:e]
			for _, f := range []uint64{5, 9} {
				for _, blk := range pbAll(inner, f) {
					urls := pbAll(blk, 1)
					hit := false
					for _, u := range urls {
						s := string(u)
						if strings.HasPrefix(s, "turn") || strings.HasPrefix(s, "stun") {
							hit = true
							break
						}
					}
					if !hit {
						continue
					}
					un, pw := pbStr(blk, 2), pbStr(blk, 3)
					for _, u := range urls {
						res = append(res, turnCred{string(u), un, pw})
					}
					for _, blk2 := range pbAll(inner, f) {
						if &blk2[0] == &blk[0] {
							continue
						}
						u2, p2 := pbStr(blk2, 2), pbStr(blk2, 3)
						for _, u := range pbAll(blk2, 1) {
							res = append(res, turnCred{string(u), u2, p2})
						}
					}
					return
				}
			}
			o = e
		case 1:
			o += 8
		case 5:
			o += 4
		default:
			return
		}
	}
	return
}

func dedup(cc []turnCred) (r []turnCred) {
	seen := map[string]bool{}
	for _, c := range cc {
		k := c.URL + "|" + c.User
		if !seen[k] {
			seen[k] = true
			r = append(r, c)
		}
	}
	return
}

// ─── Tunnel ───

type cFn func()

func (f cFn) Close() error { f(); return nil }

type mClose struct{ cc []io.Closer }

func (m *mClose) Close() error {
	for _, c := range m.cc {
		c.Close()
	}
	return nil
}

func ymxCfg() *yamux.Config {
	c := yamux.DefaultConfig()
	c.EnableKeepAlive, c.KeepAliveInterval = true, 10*time.Second
	c.ConnectionWriteTimeout, c.StreamOpenTimeout = 10*time.Second, 10*time.Second
	return c
}

func dial(cred turnCred, peer, pw string) (*yamux.Session, io.Closer, error) {
	addr := strings.TrimPrefix(strings.TrimPrefix(strings.Split(cred.URL, "?")[0], "turn:"), "turns:")
	uc, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, nil, err
	}

	tc, err := turn.NewClient(&turn.ClientConfig{STUNServerAddr: addr, TURNServerAddr: addr, Conn: uc, Username: cred.User, Password: cred.Pass})
	if err != nil {
		uc.Close()
		return nil, nil, err
	}
	if err := tc.Listen(); err != nil {
		tc.Close()
		uc.Close()
		return nil, nil, err
	}
	relay, err := tc.Allocate()
	if err != nil {
		tc.Close()
		uc.Close()
		return nil, nil, err
	}

	blk, _ := kcp.NewAESBlockCrypt(key(pw))
	kc, err := kcp.NewConn(peer, blk, 10, 3, relay)
	if err != nil {
		tc.Close()
		uc.Close()
		return nil, nil, err
	}
	kc.SetNoDelay(1, 10, 2, 1)
	kc.SetWindowSize(1024, 1024)
	kc.SetStreamMode(true)

	ym, err := yamux.Client(kc, ymxCfg())
	if err != nil {
		kc.Close()
		tc.Close()
		uc.Close()
		return nil, nil, err
	}
	return ym, &mClose{[]io.Closer{ym, kc, cFn(func() { tc.Close() }), uc}}, nil
}

// ─── Session ───

type sess struct {
	sync.RWMutex
	ym *yamux.Session
	cl io.Closer
	ok bool
}

func (s *sess) set(y *yamux.Session, c io.Closer) {
	s.Lock()
	defer s.Unlock()
	if s.cl != nil {
		s.cl.Close()
	}
	s.ym, s.cl, s.ok = y, c, true
}
func (s *sess) get() (*yamux.Session, bool) { s.RLock(); defer s.RUnlock(); return s.ym, s.ok }
func (s *sess) down()                       { s.Lock(); defer s.Unlock(); s.ok = false }
func (s *sess) stop() {
	s.Lock()
	defer s.Unlock()
	if s.cl != nil {
		s.cl.Close()
		s.cl = nil
	}
	s.ym, s.ok = nil, false
}

func establish(cache *credsCache, peer, pw string, force bool) (*yamux.Session, io.Closer, error) {
	creds, err := cache.get(force)
	if err != nil {
		return nil, nil, err
	}

	n := 0
	for _, c := range creds {
		if strings.HasPrefix(c.URL, "turn") {
			n++
		}
	}

	var last error
	i := 0
	for _, c := range creds {
		if !strings.HasPrefix(c.URL, "turn") {
			continue
		}
		i++
		sp := spinner(fmt.Sprintf("TURN %d/%d: %s", i, n, c.URL))
		ym, cl, err := dial(c, peer, pw)
		if err == nil {
			ch := make(chan error, 1)
			go func() { _, e := ym.Ping(); ch <- e }()
			select {
			case e := <-ch:
				sp.done()
				if e == nil {
					return ym, cl, nil
				}
				last = e
			case <-time.After(5 * time.Second):
				sp.done()
				last = fmt.Errorf("timeout")
			}
			cl.Close()
		} else {
			sp.done()
			last = err
		}
		wrn("  %v", last)
	}
	return nil, nil, fmt.Errorf("все TURN недоступны: %v", last)
}

// ─── Client ───

func runClient(ctx context.Context, peer, pw string) {
	killSiblings()

	cache, st, rch := &credsCache{}, &sess{}, make(chan struct{}, 1)

	ym, cl, err := establish(cache, peer, pw, true)
	if err != nil {
		die("Туннель: %v", err)
	}
	st.set(ym, cl)

	// health
	go func() {
		tk := time.NewTicker(healthEvery)
		defer tk.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				if y, ok := st.get(); ok && y != nil {
					if _, e := y.Ping(); e != nil {
						wrn("Связь потеряна")
						st.down()
						select {
						case rch <- struct{}{}:
						default:
						}
					}
				}
			}
		}
	}()

	// reconnect
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-rch:
				bo := 2 * time.Second
				for a := 1; ; a++ {
					select {
					case <-ctx.Done():
						return
					default:
					}
					inf("Переподключение (#%d)...", a)
					y, c, e := establish(cache, peer, pw, a > 3)
					if e == nil {
						st.set(y, c)
						inf("Связь восстановлена!")
						break
					}
					select {
					case <-ctx.Done():
						return
					case <-time.After(bo):
					}
					if bo *= 2; bo > maxBackoff {
						bo = maxBackoff
					}
				}
			}
		}
	}()

	l, err := net.Listen("tcp", "0.0.0.0:1080")
	if err != nil {
		die("Порт 1080: %v", err)
	}
	go func() { <-ctx.Done(); l.Close() }()

	fmt.Println()
	inf("   Туннель активен!")
	inf("   Этот ПК:   \033[32m127.0.0.1:1080\033[0m")
	inf("   Телефон:   \033[32m%s:1080\033[0m", localIP())
	inf("   Ctrl+C — выход")
	fmt.Println()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				st.stop()
				return
			default:
				continue
			}
		}
		go func(c net.Conn) {
			defer c.Close()
			y, ok := st.get()
			if !ok || y == nil {
				return
			}
			s, err := y.OpenStream()
			if err != nil {
				st.down()
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

	blk, _ := kcp.NewAESBlockCrypt(key(pw))
	l, err := kcp.ListenWithOptions(addr, blk, 10, 3)
	if err != nil {
		die("KCP: %v", err)
	}

	inf("Сервер: \033[32m%s\033[0m", addr)

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
			ym, err := yamux.Server(s, ymxCfg())
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
	fmt.Print(banner)

	// Instant Ctrl+C: force exit if pressed twice
	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		fmt.Println()
		inf("Выход...")
		cancel()
		go func() { time.Sleep(2 * time.Second); os.Exit(0) }()
		<-sig // second Ctrl+C = instant kill
		os.Exit(0)
	}()

	cfg := loadCfg()

	// Auto-update systemd service if running manually
	if runtime.GOOS == "linux" {
		replaceService()
	}

	if cfg.Role == "server" {
		runServer(ctx, cfg.ServerListen, cfg.Password)
	} else {
		runClient(ctx, cfg.ClientPeer, cfg.Password)
	}
}
