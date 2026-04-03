package golib

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Функция обращается к GitHub API с самого iPhone (это решает проблему с 137 на сервере)
func getLatestReleaseURL(arch string) (string, error) {
	resp, err := http.Get("https://api.github.com/repos/jaykaiperson/lionheart/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	assets, ok := result["assets"].([]interface{})
	if !ok {
		return "", fmt.Errorf("no assets found")
	}

	for _, a := range assets {
		asset := a.(map[string]interface{})
		url := asset["browser_download_url"].(string)
		if strings.Contains(url, "linux-"+arch) {
			return url, nil
		}
	}
	return "", fmt.Errorf("architecture %s not found", arch)
}

func generatePassword() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func extractPassword(config string) string {
	if !strings.Contains(config, `"Password"`) {
		return ""
	}
	parts := strings.Split(config, `"Password":"`)
	if len(parts) > 1 {
		return strings.Split(parts[1], `"`)[0]
	}
	return ""
}

func runCmd(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.CombinedOutput(cmd)
	return string(out), err
}

func InstallServer(host string, port int, user string, pass string) (string, error) {
	target := fmt.Sprintf("%s:%d", host, port)
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
				answers = make([]string, len(questions))
				for i := range answers {
					answers[i] = pass
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	client, err := ssh.Dial("tcp", target, config)
	if err != nil {
		return "", fmt.Errorf("ssh failed: %v", err)
	}
	defer client.Close()

	// Шаг 1: Архитектура
	archStr, _ := runCmd(client, "uname -m")
	dlArch := "x64"
	if strings.Contains(archStr, "aarch") || strings.Contains(archStr, "arm") {
		dlArch = "arm64"
	}

	// Шаг 2: Получаем URL (iPhone скачивает JSON с GitHub, сервер отдыхает)
	dlURL, err := getLatestReleaseURL(dlArch)
	if err != nil {
		return "", fmt.Errorf("API Error: %v", err)
	}

	// Шаг 3: Останавливаем старый сервер
	runCmd(client, "systemctl stop lionheart.service 2>/dev/null")

	// Шаг 4: Конфиг и пароль
	runCmd(client, "mkdir -p /opt/lionheart")
	cfgOut, _ := runCmd(client, "cat /opt/lionheart/config.json 2>/dev/null")
	password := extractPassword(cfgOut)
	if password == "" {
		password = generatePassword()
		newCfg := fmt.Sprintf(`{"Role":"server","Password":"%s","ServerListen":"0.0.0.0:8443","ClientPeer":""}`, password)
		runCmd(client, fmt.Sprintf(`echo '%s' > /opt/lionheart/config.json`, newCfg))
	}

	// Шаг 5: Получаем IP
	ip, _ := runCmd(client, "curl -s --max-time 5 https://api.ipify.org")
	ip = strings.TrimSpace(ip)
	if !strings.Contains(ip, ".") {
		ip = host
	}

	// Шаг 6: Скачиваем бинарник
	dlCmd := fmt.Sprintf("curl -fsSL --max-time 60 -L '%s' -o /opt/lionheart/lionheart.new && chmod +x /opt/lionheart/lionheart.new && mv /opt/lionheart/lionheart.new /opt/lionheart/lionheart", dlURL)
	out, err := runCmd(client, dlCmd)
	if err != nil {
		return "", fmt.Errorf("Download error: %v | Log: %s", err, out)
	}

	// Шаг 7: Служба Systemd
	service := `[Unit]
Description=Lionheart VPN Server
After=network.target
[Service]
Type=simple
User=root
WorkingDirectory=/opt/lionheart
ExecStart=/opt/lionheart/lionheart
Restart=on-failure
RestartSec=5
[Install]
WantedBy=multi-user.target`
	
	runCmd(client, fmt.Sprintf("cat << 'EOF' > /etc/systemd/system/lionheart.service\n%s\nEOF", service))
	runCmd(client, "systemctl daemon-reload && systemctl enable lionheart.service && systemctl start lionheart.service")

	// Шаг 8: Проверка запуска
	time.Sleep(2 * time.Second)
	status, _ := runCmd(client, "systemctl is-active lionheart.service")
	if strings.TrimSpace(status) != "active" {
		logs, _ := runCmd(client, "journalctl -u lionheart.service --no-pager -n 10")
		return "", fmt.Errorf("Сервер не запустился: %s\nLogs: %s", status, logs)
	}

	// Шаг 9: Возвращаем смарт-ключ
	raw := fmt.Sprintf("%s:8443|%s", ip, password)
	smartKey := base64.RawURLEncoding.EncodeToString([]byte(raw))

	version, _ := runCmd(client, "journalctl -u lionheart.service --no-pager -n 20 2>/dev/null | grep -oP 'v\\K[0-9]+\\.[0-9]+' | tail -1")
	version = strings.TrimSpace(version)
	if version == "" {
		version = "installed"
	}

	return smartKey + "|||" + version, nil
}

func UpdateServer(host string, port int, user string, pass string) (string, error) {
	target := fmt.Sprintf("%s:%d", host, port)
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
				answers = make([]string, len(questions))
				for i := range answers {
					answers[i] = pass
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	client, err := ssh.Dial("tcp", target, config)
	if err != nil {
		return "", err
	}
	defer client.Close()

	archStr, _ := runCmd(client, "uname -m")
	dlArch := "x64"
	if strings.Contains(archStr, "aarch") || strings.Contains(archStr, "arm") {
		dlArch = "arm64"
	}

	dlURL, err := getLatestReleaseURL(dlArch)
	if err != nil {
		return "", err
	}

	runCmd(client, "systemctl stop lionheart.service 2>/dev/null")

	dlCmd := fmt.Sprintf("curl -fsSL --max-time 60 -L '%s' -o /opt/lionheart/lionheart.new && chmod +x /opt/lionheart/lionheart.new && mv /opt/lionheart/lionheart.new /opt/lionheart/lionheart", dlURL)
	out, err := runCmd(client, dlCmd)
	if err != nil {
		return "", fmt.Errorf("Download error: %v | Log: %s", err, out)
	}

	runCmd(client, "systemctl start lionheart.service")
	time.Sleep(2 * time.Second)

	version, _ := runCmd(client, "journalctl -u lionheart.service --no-pager -n 20 2>/dev/null | grep -oP 'v\\K[0-9]+\\.[0-9]+' | tail -1")
	version = strings.TrimSpace(version)
	if version == "" {
		version = "updated"
	}

	return version, nil
}