// lionheart — SOCKS5 over KCP via WB Stream TURN (Fast & Verbose)
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/armon/go-socks5"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/hashicorp/yamux"
	"github.com/pion/turn/v4"
	"github.com/xtaci/kcp-go/v5"
)

const configFile = "config.json"

type Config struct {
	Role         string `json:"role"`
	Password     string `json:"password"`
	ServerListen string `json:"server_listen"`
	ClientPeer   string `json:"client_peer"`
}

func fatal(format string, v ...any) {
	fmt.Printf("\n[КРИТИЧЕСКАЯ ОШИБКА] "+format+"\nНажмите Enter для выхода...\n", v...)
	bufio.NewReader(os.Stdin).ReadString('\n')
	os.Exit(1)
}

func readInput(prompt string) string {
	fmt.Print(prompt)
	val, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(val)
}

func getPublicIP() string {
	resp, err := http.Get("https://api.ipify.org")
	if err == nil && resp != nil {
		defer resp.Body.Close()
		if b, err := io.ReadAll(resp.Body); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

func installSystemdService() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Ошибка: не удалось определить путь к файлу: %v\n", err)
		return
	}
	absPath, _ := filepath.Abs(exePath)
	workDir := filepath.Dir(absPath)

	serviceStr := fmt.Sprintf(`[Unit]
Description=Lionheart Proxy Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=%s
ExecStart=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target`, workDir, absPath)

	err = os.WriteFile("/etc/systemd/system/lionheart.service", []byte(serviceStr), 0644)
	if err != nil {
		fmt.Printf("\n[!] Ошибка создания службы (%v). Запустите программу через sudo.\n", err)
		return
	}

	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "lionheart.service").Run()
	exec.Command("systemctl", "start", "lionheart.service").Run()

	fmt.Println("\n✅ Служба успешно установлена и запущена в фоне!")
	os.Exit(0)
}

func runWizard() *Config {
	fmt.Println("Настройка lionheart")
	cfg := &Config{}

	if readInput("Режим (1 - сервер, 2 - клиент): ") == "1" {
		cfg.Role, cfg.ServerListen = "server", "0.0.0.0:56000"
		b := make([]byte, 16)
		rand.Read(b)
		cfg.Password = hex.EncodeToString(b)

		fmt.Print("Определение публичного IP сервера...")
		ip := getPublicIP()
		fmt.Println(" Готово.")

		if ip == "" || !strings.Contains(ip, ".") {
			ip = readInput("Не удалось определить IP. Введите IP сервера вручную: ")
		}

		rawKey := fmt.Sprintf("%s|%s", ip, cfg.Password)
		inviteKey := base64.RawURLEncoding.EncodeToString([]byte(rawKey))

		fmt.Printf("\n--- ВАЖНО ---\nВаш СМАРТ-КЛЮЧ для клиента:\n%s\nСкопируйте его.\n-------------\n\n", inviteKey)

		data, _ := json.MarshalIndent(cfg, "", "  ")
		os.WriteFile(configFile, data, 0644)

		if runtime.GOOS == "linux" && readInput("Установить как службу (работа в фоне)? (y/n): ") == "y" {
			installSystemdService()
		}
		readInput("Нажмите Enter для запуска сервера в текущем окне...")

	} else {
		cfg.Role = "client"
		inviteKey := readInput("Введите смарт-ключ от сервера: ")

		decoded, err := base64.RawURLEncoding.DecodeString(inviteKey)
		if err != nil {
			fatal("Неверный формат смарт-ключа: %v", err)
		}

		parts := strings.SplitN(string(decoded), "|", 2)
		if len(parts) != 2 {
			fatal("Поврежденный смарт-ключ (неверное количество частей)")
		}

		cfg.ClientPeer = parts[0]
		if !strings.Contains(cfg.ClientPeer, ":") {
			cfg.ClientPeer += ":56000"
		}
		cfg.Password = parts[1]

		data, _ := json.MarshalIndent(cfg, "", "  ")
		os.WriteFile(configFile, data, 0644)
		fmt.Println("Готово. Настройки клиента сохранены.")
	}

	return cfg
}

func loadConfig() *Config {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return runWizard()
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		fatal("Не удалось прочитать config.json: %v", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fatal("Ошибка парсинга config.json: %v", err)
	}
	return &cfg
}

type turnCreds struct {
	URL, User, Pass string
}

const interceptJS = `
Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
window._tc=[];
const _O=window.RTCPeerConnection;
window.RTCPeerConnection=function(...a){
	const c=a[0]||{};
	(c.iceServers||[]).forEach(s=>{
		const us=Array.isArray(s.urls)?s.urls:(s.urls?[s.urls]:[]);
		us.forEach(u=>{
			if(u&&u.indexOf('turn')===0) window._tc.push({url:u,user:s.username||'',pass:s.credential||''});
		});
	});
	return new _O(...a);
};
Object.keys(_O).forEach(k=>{try{window.RTCPeerConnection[k]=_O[k]}catch(e){}});
window.RTCPeerConnection.prototype=_O.prototype;

setInterval(()=>{
	try{
		const txt = (el) => (el.innerText || el.textContent || '').toLowerCase().replace(/\s+/g, ' ').trim();
		const els = Array.from(document.querySelectorAll('*')).filter(el => el.offsetHeight > 0);
		
		const btns = els.filter(el => ['BUTTON', 'A'].includes(el.tagName) || el.getAttribute('role') === 'button');
		
		const goMain = btns.find(b => txt(b).includes('main page'));
		if(goMain) { goMain.click(); return; }

		const guest = btns.find(b => txt(b).includes('guest'));
		if(guest) { guest.click(); return; }

		const inp = document.querySelector('input[type="text"], input[placeholder*="name" i]');
		if(inp && !inp.value){
			document.querySelectorAll('input[type="checkbox"]').forEach(c => {
				if(!c.checked){ c.click(); c.dispatchEvent(new Event('change',{bubbles:true})); }
			});
			const s = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value');
			if(s && s.set) { s.set.call(inp, 'lionheart'); inp.dispatchEvent(new Event('input',{bubbles:true})); }
			else { inp.value = 'lionheart'; }
			return;
		}

		const cont = btns.find(b => txt(b) === 'continue' || txt(b) === 'join' || txt(b).includes('join meeting'));
		if(cont) { cont.click(); return; }

		const meetingBlock = els.filter(el => 
			el.tagName === 'DIV' && txt(el).includes('new video meeting') && el.children.length < 5
		).sort((a, b) => a.innerText.length - b.innerText.length)[0];
		
		if(meetingBlock) { meetingBlock.click(); }

	} catch(e){}
}, 200); 
`

func getTurnCreds() ([]turnCreds, error) {
	fmt.Print("Получение TURN-маршрута платформы...")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"), chromedp.Flag("mute-audio", true),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.Flag("lang", "en-US"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, 40*time.Second)
	defer cancelTimeout()

	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		network.Enable().Do(ctx)
		network.SetExtraHTTPHeaders(network.Headers{"Accept-Language": "en-US,en;q=0.9"}).Do(ctx)
		_, err := page.AddScriptToEvaluateOnNewDocument(interceptJS).Do(ctx)
		return err
	}))

	if err != nil {
		return nil, fmt.Errorf("ошибка инъекции JS-автопилота: %v", err)
	}

	if err := chromedp.Run(timeoutCtx, chromedp.Navigate("https://stream.wb.ru/")); err != nil {
		return nil, fmt.Errorf("ошибка перехода на сайт платформы: %v", err)
	}

	for i := 0; i < 100; i++ {
		var raw string
		err := chromedp.Run(timeoutCtx, chromedp.Evaluate(`JSON.stringify(window._tc || [])`, &raw))

		if err == nil && raw != "[]" && raw != "" {
			var creds []turnCreds
			if err := json.Unmarshal([]byte(raw), &creds); err == nil && len(creds) > 0 {
				fmt.Println(" Успешно.")
				return creds, nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return nil, fmt.Errorf("превышено время ожидания (таймаут платформы или блокировка интерфейса)")
}

func runServer(listenAddr, password string) {
	key := sha256.Sum256([]byte(password))
	block, _ := kcp.NewAESBlockCrypt(key[:])

	l, err := kcp.ListenWithOptions(listenAddr, block, 10, 3)
	if err != nil {
		fatal("Ошибка запуска KCP сервера: %v", err)
	}
	fmt.Printf("Сервер успешно запущен и слушает %s\n", listenAddr)

	socksSrv, err := socks5.New(&socks5.Config{})
	if err != nil {
		fatal("Ошибка создания SOCKS5 сервера: %v", err)
	}

	for {
		s, err := l.AcceptKCP()
		if err != nil {
			continue
		}

		s.SetNoDelay(1, 10, 2, 1)
		s.SetWindowSize(1024, 1024)
		s.SetStreamMode(true)

		go func(session *kcp.UDPSession) {
			ymx, err := yamux.Server(session, nil)
			if err != nil {
				session.Close()
				return
			}
			defer ymx.Close()
			for {
				stream, err := ymx.AcceptStream()
				if err != nil {
					return
				}
				go socksSrv.ServeConn(stream)
			}
		}(s)
	}
}

func connectTurn(creds turnCreds, peer, password string) (*yamux.Session, error) {
	addr := strings.TrimPrefix(strings.TrimPrefix(strings.Split(creds.URL, "?")[0], "turn:"), "turns:")

	// Открываем свободный, непривязанный UDP-порт (решает ошибку pre-connected)
	uConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("локальный UDP порт недоступен: %v", err)
	}

	client, err := turn.NewClient(&turn.ClientConfig{
		STUNServerAddr: addr, TURNServerAddr: addr,
		Conn: uConn, Username: creds.User, Password: creds.Pass,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации TURN клиента: %v", err)
	}

	if err := client.Listen(); err != nil {
		return nil, fmt.Errorf("ошибка Listen TURN: %v", err)
	}

	relay, err := client.Allocate()
	if err != nil {
		return nil, fmt.Errorf("сервер отказал в выделении (Allocate) порта: %v", err)
	}

	key := sha256.Sum256([]byte(password))
	block, _ := kcp.NewAESBlockCrypt(key[:])

	kConn, err := kcp.NewConn(peer, block, 10, 3, relay)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания KCP соединения: %v", err)
	}

	kConn.SetNoDelay(1, 10, 2, 1)
	kConn.SetWindowSize(1024, 1024)
	kConn.SetStreamMode(true)

	ymx, err := yamux.Client(kConn, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка мультиплексора Yamux: %v", err)
	}

	return ymx, nil
}

func getLocalIP() string {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ip, ok := a.(*net.IPNet); ok && !ip.IP.IsLoopback() && ip.IP.To4() != nil {
			return ip.IP.String()
		}
	}
	return "Ваш_Локальный_IP"
}

func runClient(peerAddr, password string) {
	creds, err := getTurnCreds()
	if err != nil || len(creds) == 0 {
		fatal("Не удалось пробиться через платформу. Детали: %v", err)
	}

	fmt.Print("Установка защищенного туннеля...")
	var ymx *yamux.Session
	var lastErr error

	for _, c := range creds {
		ymx, lastErr = connectTurn(c, peerAddr, password)
		if lastErr == nil && ymx != nil {
			break
		}
	}

	if ymx == nil {
		fatal("Туннель не установлен (платформа выдала мертвые сервера). Последняя ошибка: %v", lastErr)
	}
	fmt.Println(" Готово.")

	l, err := net.Listen("tcp", "0.0.0.0:1080")
	if err != nil {
		fatal("Не удалось открыть локальный порт 1080 (возможно он занят другой программой): %v", err)
	}

	fmt.Printf("\nТуннель успешно запущен!\n")
	fmt.Printf("Прокси для этого ПК: 127.0.0.1:1080\n")
	fmt.Printf("Прокси для телефона: %s:1080\n", getLocalIP())
	fmt.Printf("\nДля остановки нажмите Ctrl+C\n")

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			s, err := ymx.OpenStream()
			if err != nil {
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

func main() {
	cfg := loadConfig()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		fmt.Print("\r\033[KЗавершение работы...\n")
		os.Exit(0)
	}()

	if cfg.Role == "server" {
		runServer(cfg.ServerListen, cfg.Password)
	} else {
		runClient(cfg.ClientPeer, cfg.Password)
	}
}
