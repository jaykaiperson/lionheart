# Миграция Lionheart на sing-box

Этот документ описывает миграцию Lionheart VPN на ядро sing-box с поддержкой правил маршрутизации.

## Что нового в v1.4

### Основные изменения

1. **Интеграция sing-box** — новое ядро на базе sing-box с поддержкой:
   - TUN интерфейса (системный VPN)
   - SOCKS5 прокси
   - Правил маршрутизации (GeoIP, GeoSite, домены, IP)
   - DNS routing
   - Ad blocking

2. **Правила маршрутизации** — гибкая система маршрутизации трафика:
   - GeoIP (по странам)
   - GeoSite (по категориям сайтов)
   - Домены
   - IP-адреса и CIDR
   - Порты
   - Протоколы

3. **Предустановленные профили** — готовые конфигурации для разных сценариев:
   - `direct_cn` — прямое соединение для китайских сайтов
   - `adblock` — блокировка рекламы
   - `streaming` — оптимизация для стриминговых сервисов
   - `gaming` — низкая задержка для игр
   - `privacy` — весь трафик через прокси
   - `russia` — оптимизация для российских пользователей
   - `belarus` — оптимизация для белорусских пользователей

4. **Обратная совместимость** — сохранена поддержка legacy KCP туннеля

## Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                      Lionheart v1.4                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   Legacy    │    │   sing-box  │    │   Tunnel    │     │
│  │    KCP      │◄──►│   Engine    │◄──►│   Manager   │     │
│  │  (Yamux)    │    │             │    │             │     │
│  └─────────────┘    └──────┬──────┘    └─────────────┘     │
│                             │                                │
│                    ┌────────┴────────┐                       │
│                    │  Routing Rules  │                       │
│                    │  (GeoIP/GeoSite)│                       │
│                    └─────────────────┘                       │
├─────────────────────────────────────────────────────────────┤
│  TURN/ICE Relay (WB Stream) — маскировка под видеозвонки    │
└─────────────────────────────────────────────────────────────┘
```

## Использование

### CLI

#### Режим сервера (без изменений)

```bash
./lionheart
# Выберите режим 1 (server)
```

#### Режим клиента с legacy KCP

```bash
./lionheart
# Выберите режим 2 (client)
# Введите Smart Key
```

#### Режим клиента с sing-box

```bash
./lionheart
# Выберите режим 3 (client+sing-box)
# Введите Smart Key
# Выберите профиль маршрутизации
```

### Конфигурационный файл

```json
{
  "role": "client",
  "client_peer": "1.2.3.4:8443",
  "password": "...",
  "use_singbox": true,
  "routing_preset": "russia",
  "routing_rules": {
    "geoip_direct": ["ru", "private"],
    "geosite_direct": ["ru", "yandex", "vk"],
    "geosite_proxy": ["twitter", "facebook", "instagram"],
    "geosite_block": ["category-ads-all"],
    "final": "proxy"
  }
}
```

### Android

```kotlin
// Использование sing-box
val vpn = LionheartVPN.getInstance()
vpn.setStatusCallback(callback)
vpn.configure(smartKey)
vpn.enableSingBox(true)
vpn.setRoutingPreset("russia")
vpn.connect()
```

## Профили маршрутизации

### direct_cn

Прямое соединение для китайских сайтов, прокси для остального.

```json
{
  "geosite_direct": ["cn", "private", "apple-cn", "microsoft-cn"],
  "geoip_direct": ["cn", "private"],
  "final": "proxy"
}
```

### adblock

Блокировка рекламы и трекеров.

```json
{
  "geosite_block": [
    "category-ads-all",
    "category-tracker",
    "category-malware"
  ],
  "final": "proxy"
}
```

### streaming

Оптимизация для стриминговых сервисов.

```json
{
  "geosite_proxy": ["netflix", "disney", "hulu", "hbo", "youtube"],
  "final": "direct"
}
```

### gaming

Низкая задержка для игр.

```json
{
  "geosite_direct": ["steam", "epicgames", "blizzard", "xbox"],
  "port_direct": [27015, 3074, 7777, 25565],
  "protocol_direct": ["udp"],
  "final": "proxy"
}
```

### russia

Оптимизация для российских пользователей.

```json
{
  "geosite_direct": [
    "ru", "yandex", "vk", "mailru",
    "sberbank", "tinkoff", "ozon", "wildberries"
  ],
  "geoip_direct": ["ru", "private"],
  "geosite_proxy": [
    "twitter", "facebook", "instagram",
    "discord", "telegram", "wikipedia"
  ],
  "final": "proxy"
}
```

## Технические детали

### Структура проекта

```
lionheart-singbox/
├── core/                      # Ядро с sing-box интеграцией
│   ├── go.mod
│   ├── tunnel.go             # Туннель с поддержкой sing-box
│   ├── singbox_config.go     # Конфигурация sing-box
│   ├── singbox_transport.go  # Транспорт Lionheart для sing-box
│   ├── routing_presets.go    # Предустановленные профили
│   ├── wb.go                 # TURN/ICE (без изменений)
│   └── protobuf.go           # Protobuf parser (без изменений)
├── cmd/lionheart/            # CLI клиент
│   ├── main.go               # Обновленный main с sing-box
│   └── go.mod
├── mobile/android/golib/     # Android библиотека
│   ├── liblionheart.go       # Обновленные биндинги
│   ├── engine.go             # VPN engine
│   └── go.mod
└── docs/
    └── SINGBOX_MIGRATION.md  # Этот документ
```

### Интеграция с sing-box

1. **Конфигурация** — `SingBoxConfig` структура с полной поддержкой sing-box options
2. **Транспорт** — `LionheartTransport` реализует интерфейс sing-box outbound
3. **Правила** — `RoutingRules` с маппингом на sing-box route rules
4. **Пресеты** — `RoutingPreset` для быстрого выбора конфигурации

### Обратная совместимость

- Legacy KCP туннель сохранен в `core/tunnel.go`
- Smart Key формат не изменен
- Серверная часть без изменений
- Android API совместим

## Миграция

### Для пользователей

1. Обновите клиент до v1.4
2. В режиме клиента выберите опцию 3 для sing-box
3. Выберите подходящий профиль маршрутизации

### Для разработчиков

1. Замените `core` пакет на новую версию
2. Обновите вызовы API для поддержки sing-box
3. Добавьте UI для выбора профилей маршрутизации

## Преимущества sing-box

1. **Производительность** — оптимизированный TUN драйвер
2. **Гибкость** — сложные правила маршрутизации
3. **Совместимость** — работает с любыми приложениями
4. **DNS routing** — продвинутая маршрутизация DNS
5. **Ad blocking** — встроенная блокировка рекламы
6. **Обновления** — активная разработка sing-box

## Ограничения

1. Увеличенный размер бинарника (~5-10 МБ)
2. Более сложная конфигурация
3. Требуется Android 7.0+ для TUN

## Дальнейшее развитие

- [ ] Поддержка WireGuard
- [ ] Поддержка Shadowsocks
- [ ] Поддержка Trojan
- [ ] Web UI для конфигурации
- [ ] Импорт/экспорт конфигураций
- [ ] Автоматический выбор сервера
