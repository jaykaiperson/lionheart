# lionheart — прокси, использующий WB Stream (платформа Wildberries) для передачи трафика.
## По случайному стечению обстоятельств обходит белые списки.

<p align="center">
  <a href="#russian">🇷🇺 Русский</a> • 
  <a href="#english">🇺🇸 English</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-blue?logo=go">
  <img src="https://img.shields.io/badge/License-GPL--3.0-green">
  <img src="https://img.shields.io/badge/status-experimental-red">
</p>

---

<a id="russian"></a>

## 🇷🇺 Русский

### Дисклеймер (отказ от ответственности)
Этот проект написан исключительно в образовательных целях. По случайному стечению обстоятельств позже выяснилось, что этот код собирается в SOCKS5-туннель, который прошивает DPI, корпоративные файрволы и мобильные белые списки.

Я категорически осуждаю использование инструмента для обхода любых сетевых ограничений. Тот факт, что программа идеально справляется с этой задачей, утилизируя гигабитные каналы стриминговых платформ — побочный эффект архитектуры.

Пожалуйста, используйте код только для изучения: не компилируйте это, не ставьте на свой VPS и не используйте для обхода блокировок. Вы предупреждены.

lionheart — это экспериментальный SOCKS5-прокси, который заворачивает трафик в протокол KCP и прячет его внутри легитимных WebRTC TURN-сессий. Проект написан на Go и использует инфраструктуру стриминговых платформ (в данном случае WB Stream) как бесплатные релеи для обхода DPI.

### Как это работает под капотом
Архитектура состоит из сервера и клиента, но они не подключаются друг к другу напрямую. Весь трафик идет транзитом через серверы платформы.

На стороне клиента под капотом через chromedp запускается скрытый экземпляр Chrome. Автопилот на JS заходит на платформу, авторизуется гостем и создает новую комнату для видеоконференции. В этот момент на страницу инжектится скрипт, который переопределяет нативный объект RTCPeerConnection.

Когда платформа пытается инициализировать WebRTC-соединение для видеосвязи, скрипт перехватывает массив iceServers. Из него извлекаются динамические адреса TURN-серверов, логины и пароли, сгенерированные платформой для этой конкретной сессии.

Имея учетные данные TURN, клиент (с помощью библиотеки pion/turn) авторизуется на сервере платформы и запрашивает UDP-порт для ретрансляции (Allocate).

Через этот порт устанавливается соединение с нашим VPS-сервером по протоколу KCP. KCP обеспечивает надежную доставку пакетов поверх UDP и шифрование (AES).

Полученный KCP-канал мультиплексируется с помощью yamux, чтобы мы могли гонять множество независимых TCP-соединений внутри одного потока. На стороне сервера yamux-сессии передаются во встроенный SOCKS5-сервер, который выпускает трафик в интернет.

### Сборка
Вам понадобится установленный Go версии 1.22 или выше. Также на машине-клиенте должен быть установлен Google Chrome или Chromium, так как он нужен для работы headless-браузера.

Перед сборкой обновите зависимости:
```
go mod tidy
```

Сборка под вашу текущую платформу:
```
go build -o lionheart main.go
```

### Использование
При первом запуске бинарника откроется консольный мастер настройки, который сформирует файл config.json.

#### Настройка сервера (VPS)
Запустите lionheart на вашем сервере и выберите режим сервера. Программа автоматически определит внешний IP-адрес сервера и сгенерирует криптографический ключ.

На выходе вы получите Base64-строку (смарт-ключ), в которой зашит IP и пароль. Этот ключ нужно будет передать клиенту.

На Linux-серверах программа предложит автоматически прописаться в systemd, чтобы работать в фоне и стартовать при перезагрузке системы.

#### Настройка клиента
Запустите lionheart на локальном ПК, выберите режим клиента и вставьте смарт-ключ, полученный на предыдущем шаге.

Программа поднимет фоновый браузер, получит маршруты, установит туннель и откроет локальный порт.

После успешного коннекта в консоли появится адрес локального SOCKS5-прокси (по умолчанию это 127.0.0.1:1080).

Теперь вы можете прописать этот адрес в настройках прокси вашего браузера, Telegram или всей системы. Если вы хотите подключить к прокси телефон, убедитесь, что ПК и телефон находятся в одной локальной сети (Wi-Fi), и используйте локальный IP-адрес ПК, который программа выведет в консоль.

### Благодарности
Этот проект был вдохновлен концепциями и исследованиями, представленными в репозитории vk-turn-proxy. Огромное спасибо автору за заложенный фундамент в техниках перехвата WebRTC TURN.

### Вклад и Лицензия
Буду рад вашим Pull Requests и Issues.

Проект распространяется по лицензии GPL-3.0 — подробности смотрите в файле LICENSE.

---

<a id="english"></a>

## 🇺🇸 English

### Disclaimer
This project is written exclusively for educational purposes. By a complete coincidence, it was later discovered that this code compiles into a SOCKS5 tunnel that pierces through DPI, corporate firewalls, and mobile whitelists.

I strictly condemn the use of this tool to bypass any network restrictions. The fact that the program perfectly handles this task, utilizing gigabit channels of streaming platforms, is merely a side effect of its architecture.

Please use this code only for learning: do not compile it, do not deploy it to your VPS, and do not use it to bypass blocks. You have been warned.

lionheart is an experimental SOCKS5 proxy that encapsulates traffic into the KCP protocol and hides it inside legitimate WebRTC TURN sessions. The project is written in Go and uses the infrastructure of streaming platforms (in this case, WB Stream) as free relays to bypass Deep Packet Inspection (DPI).

### How it works under the hood
The architecture consists of a server and a client, but they do not connect to each other directly. All traffic is routed through the platform's servers.

On the client side, a hidden Chrome instance is launched via chromedp. The JS autopilot navigates to the platform, logs in as a guest, and creates a new video conference room. At this point, a script is injected into the page that overrides the native RTCPeerConnection object.

When the platform tries to initialize a WebRTC connection for video communication, the script intercepts the iceServers array. Dynamic TURN server addresses, logins, and passwords generated by the platform for this specific session are extracted.

Having the TURN credentials, the client (using the pion/turn library) authenticates on the platform's server and requests a UDP port for relaying (Allocate).

Through this port, a connection is established with our VPS server via the KCP protocol. KCP provides reliable packet delivery over UDP and AES encryption.

The resulting KCP channel is multiplexed using yamux, allowing us to route multiple independent TCP connections within a single stream. On the server side, yamux sessions are passed to the built-in SOCKS5 server, which releases the traffic to the internet.

### Build Instructions
You will need Go version 1.22 or higher. Also, Google Chrome or Chromium must be installed on the client machine, as it is required for the headless browser to work.

Update dependencies before building:
```
go mod tidy
```

Build for your current platform:
```
go build -o lionheart main.go
```

### Usage
Upon the first launch of the binary, a console setup wizard will open, which will generate a config.json file.

#### Server Setup (VPS)
Run lionheart on your server and select the server mode. The program will automatically detect the external IP address of the server and generate a cryptographic key.

The output will provide a Base64 string (Smart-Key), containing the IP and password. You need to pass this key to the client.

On Linux servers, the program will offer to automatically create a systemd service to run in the background and start on reboot.

#### Client Setup
Run lionheart on your local PC, select the client mode, and paste the Smart-Key obtained in the previous step.

The program will spin up a background browser, fetch the routes, establish the tunnel, and open a local port.

After a successful connection, the local SOCKS5 proxy address will appear in the console (by default, it is 127.0.0.1:1080).

Now you can configure this address in the proxy settings of your browser, Telegram, or the entire OS. If you want to connect a smartphone to the proxy, ensure the PC and phone are on the same local network (Wi-Fi), and use the PC's local IP address displayed in the console.

### Acknowledgements
This project was heavily inspired by the concepts and research presented in vk-turn-proxy. Huge thanks to the author for laying the groundwork for WebRTC TURN hijacking techniques.

### Contributing & License
Contributions, issues, and feature requests are welcome! Feel free to check the issues page.

This project is licensed under the GPL-3.0 License - see the LICENSE file for details.
