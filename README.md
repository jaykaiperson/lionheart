<div align="center">

# lionheart

**Приватный децентрализованный self-hosted туннель**

[![Latest Release](https://img.shields.io/github/v/release/jaykaiperson/lionheart?style=flat-square&color=success)](https://github.com/jaykaiperson/lionheart/releases/latest)
![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&style=flat-square)
![Android API](https://img.shields.io/badge/Android-API_21%2B-3DDC84?logo=android&style=flat-square)
[![License](https://img.shields.io/github/license/jaykaiperson/lionheart?style=flat-square)](LICENSE)

[🇷🇺 Русский](#-русская-версия) · [🇬🇧 English](#-english-version)

</div>

---

## ⚠️ Дисклеймер / Disclaimer

> **RU:** Этот проект создан в образовательных целях — для изучения сетевых протоколов, шифрования и архитектуры P2P-туннелей. Автор призывает использовать его исключительно в законных целях и не одобряет обход государственных блокировок и белых списков. К сожалению, техническая природа проекта делает такой обход возможным — и, надо признать, весьма эффективным. Вы используете программу на свой страх и риск, в соответствии с законодательством вашей страны.
>
> **EN:** This project was built for educational purposes — to study network protocols, encryption, and P2P tunnel architecture. The author encourages use strictly within the bounds of applicable law and does not endorse circumventing government-imposed restrictions or allowlists. Unfortunately, the technical nature of the project makes such circumvention possible — and, to be frank, quite effective. You use this software at your own risk and in accordance with the laws of your jurisdiction.

---

## 🇷🇺 Русская версия

### 🚀 Что нового в v1.3

> **SOCKS5 без перехвата системного слота** — На Android режим SOCKS5 теперь поднимается как обычный фоновый процесс (`127.0.0.1:1080`) без вызова `VpnService.Builder.establish()`. Это освобождает системный слот и позволяет использовать Lionheart параллельно с v2ray или OpenVPN.
>
> **Устранена гонка потоков в логах** — Исправлены краши `ConcurrentModificationException` при просмотре логов. Данные из фонового JNI-потока ядра Go теперь безопасно передаются в главный UI-поток через `viewModelScope.launch(Dispatchers.Main)`, исключая падения Jetpack Compose под нагрузкой.
>
> **Обход брандмауэра Windows** — Механизм получения внешнего IP переведён с «голых» TCP-сокетов на `net/http` с фоллбэком на альтернативные API. Теперь `pubIP()` корректно обходит правила Windows Defender и не возвращает `0.0.0.0`.
>
> **Улучшения UI/UX и совместимости** — Добавлено разрешение `QUERY_ALL_PACKAGES` для Android 11+ (API 30) — восстановлено отображение полного списка приложений в настройках Split Tunneling. Кнопки ручного ввода Smart Key и QR-сканер получили полноценные обработчики и навигацию.
>
> **Надёжный деплой по SSH** — Скрипт авто-установки переведён на GitHub API для динамического парсинга ссылок на свежие x64 релизы. Полностью исключены ошибки 404 при изменениях в версионировании бинарников.

### Что такое Lionheart

Lionheart — приватный децентрализованный self-hosted туннель с высокопроизводительным ядром на Go и нативным Android-клиентом. Никаких центральных серверов, «комнат» или сбора метаданных — трафик идёт строго напрямую с вашего устройства на ваш личный VPS.

Регистрация не требуется. Вы устанавливаете серверную часть на VPS, она автоматически генерирует **Smart Key** — закодированную строку с IP-адресом, портом и паролем сессии. Клиент на Android или ПК использует этот ключ для создания зашифрованного туннеля через протоколы **KCP** и **Yamux**. Ядро работает как локальный SOCKS5 прокси или как полноценный системный туннель на Android.

---

## 🇬🇧 English Version

### 🚀 What's New in v1.3

> **True SOCKS5 Proxy Mode** — On Android, SOCKS5 mode is now handled strictly as a Foreground Service binding to `127.0.0.1:1080` without invoking `VpnService.Builder.establish()`. This frees up the Android system slot, allowing Lionheart to run concurrently with other apps like v2ray or OpenVPN.
>
> **Thread-Safe Real-Time Logs** — Fixed `ConcurrentModificationException` crashes when viewing live logs. Data streams from Go core JNI threads are now safely dispatched to the Main thread (`viewModelScope.launch(Dispatchers.Main)`) before updating Jetpack Compose UI states.
>
> **Windows Firewall Evasion** — Completely rewrote the public IP detection in the CLI logic. Migrated from raw TCP sockets (often blocked by Windows Defender) to standard `net/http` clients with fallback APIs, resolving the persistent `0.0.0.0` IP bug.
>
> **UI/UX & Compatibility Polishing** — Added `QUERY_ALL_PACKAGES` permission for Android 11+ (API 30) to populate the full application list in Split Tunneling settings. Fully implemented navigation logic and dialogs for the QR code scanner and manual Smart Key entry.
>
> **Robust VPS Deployment** — The automated SSH deployment script now uses the GitHub API to dynamically parse and fetch the latest x64 release artifacts, effectively eliminating 404 errors during server setup.

### What is Lionheart

Lionheart is a private, decentralized self-hosted tunnel. It consists of a high-performance Go-based core and a native Android client. There are no central servers, third-party coordinators, or tracking mechanisms — your traffic flows directly between your device and your personal VPS.

No registration required. Deploy the server component on your VPS and it automatically generates a **Smart Key** — a base64-encoded string containing the server's IP, port, and session password. The Android or CLI client uses this key to establish a secure, encrypted tunnel via **KCP** and **Yamux**. The core can operate either as a local SOCKS5 proxy or as a full-system Android tunnel.
