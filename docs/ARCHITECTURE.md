# Архитектура 3wayproxy

## Цели

1. **Cover story**: трафик выглядит как игра в браузере (змейка), не как VPN/API-туннель.
2. **Per-relay blindness**: ни один relay не видит целый IP-пакет и не знает конечный destination.
3. **DPI churn**: WSS-соединения живут 2–5 с, затем «переподключение игрока».
4. **Logical continuity**: клиент и aggregator держат одну сессию поверх churn.
5. **Без исходящих curl с relay**: aggregator инициирует связь к relay (inbound WSS на хостинг).

## Топология

```
                    ┌─────────────────────────────────────┐
                    │           Aggregator (exit)          │
                    │  TUN · reassembly · NAT · CDN       │
                    └──────────┬──────────┬──────────┬────┘
                               │ WSS      │ WSS      │ WSS
                    (роль «matchmaker» / «spectator»)
                               │          │          │
              ┌────────────────┘          │          └────────────────┐
              ▼                             ▼                             ▼
       ┌─────────────┐              ┌─────────────┐              ┌─────────────┐
       │  Relay A    │              │  Relay B    │              │  Relay C    │
       │  Python+WSS │              │  Python+WSS │              │  Python+WSS │
       │  + игра     │              │  + игра     │              │  + игра     │
       └──────▲──────┘              └──────▲──────┘              └──────▲──────┘
              │ WSS churn                │ WSS churn                │ WSS (idle)
              │                          │                          │
              └──────────────────────────┼──────────────────────────┘
                                         │
                              ┌──────────▼──────────┐
                              │  Client             │
                              │  TUN + Chromium     │
                              │  2 active shards    │
                              └─────────────────────┘
```

## Почему WSS, а не curl с relay

Shared-хостинг часто блокирует исходящие `curl`/`fsockopen`. Зато **входящие** HTTPS и WSS — норма для веб-приложения.

**Решение**: aggregator сам подключается к каждому relay по WSS (как «второй игрок» или «зритель матча»). Relay только:

- принимает WSS от клиента (игрок);
- принимает WSS от aggregator (скрытая роль по секретному handshake);
- пересылает фрагменты между ними в RAM/кратковременном буфере.

Исходящий трафик с relay к aggregator **не нужен**.

## Почему не WebRTC (на первом этапе)

| Критерий | WSS | WebRTC |
|----------|-----|--------|
| На shared Python | реалистично (если есть ASGI/long-lived) | нужен TURN, UDP, ICE — редко на shared |
| Маскировка | обычный game sync | отдельный UDP-поток, палится иначе |
| R&D сложность | низкая | высокая |

WebRTC — **фаза 5+** (опционально) для datachannel между клиентом и aggregator **мимо** relay, если появится свой VPS. Для текущей модели «3 игры на хостинге» — **Binary WSS**.

## Ротация 2+1

В каждый момент:

- **2 relay активны** — принимают/отдают фрагменты;
- **1 relay отдыхает** ~1 с (имитация паузы между нажатиями).

Клиент ведёт таблицу:

```
slot 0: relay-A  [ACTIVE]
slot 1: relay-B  [ACTIVE]
slot 2: relay-C  [IDLE until T+1s]
```

Каждые `rotate_interval` (например 3–8 с, случайно):

1. Закрыть WSS на idle-слоте (уже закрыт) или на том, кто уходит в idle.
2. Сдвинуть роли: бывший active → idle, бывший idle → active.
3. Новый active открывает WSS с тем же `session_id` (resume handshake).

Aggregator зеркалит ту же ротацию — знает, с каких relay забирать uplink и куда слать downlink.

**Шардирование**: фрагмент IP-пакета с `frag_idx mod 2` идёт на один из двух active relay (не на idle).

## WSS churn (разрыв для DPI)

### Поведение для наблюдателя DPI

- Короткие WebSocket-сессии 2–5 с.
- Закрытие с кодом 1000/1001 («уход со страницы» / «reconnect»).
- Сразу новое подключение с тем же cookie `game_session`.
- Между reconnect — 1–3 POST/GET обычной игры (score, canvas ping) через fetch в Chromium.

### Поведение для протокола

При reconnect клиент шлёт:

```
WS_FRAME_RESUME { session_id, last_seq_ack, shard_id }
```

Relay/Aggregator восстанавливают окно reassembly из RAM (TTL 10–30 с) или из краткого store (SQLite на relay).

## Headless Chromium

### Роли

| Задача | Кто выполняет |
|--------|----------------|
| TLS fingerprint Chrome | Chromium (не curl) |
| Cookie / localStorage session | Chromium |
| Binary WSS send/recv | JS внутри страницы игры → CDP bridge → native TUN |
| Cover: движения змейки | JS autoplay + случайные паузы |
| Загрузка «библиотек» с CDN | `<script src="https://exit.example/static/jquery.min.js">` |

### Схема client

```
┌─────────────────────────────────────────┐
│ 3wayproxy-client (Go или Rust)          │
│  ├─ TUN reader/writer                   │
│  ├─ Fragmenter / Reassembler           │
│  ├─ Playwright/CDP controller          │
│  └─ 3× browser context (или 1 + 3 tab)│
└─────────────────────────────────────────┘
         │ CDP Runtime.evaluate / exposeBinding
         ▼
┌─────────────────────────────────────────┐
│ game/snake.js + carrier/ws_carrier.js   │
│  ├─ canvas game (cover)                  │
│  ├─ WebSocket → relay                   │
│  └─ window.__carrier.send(chunk)        │
└─────────────────────────────────────────┘
```

Native код не открывает WSS напрямую — только через Chromium, чтобы JA3 и HTTP/2 совпадали с игрой.

## CDN-библиотека с aggregator

Страница игры подключает:

```html
<script src="https://cdn.exit.example/libs/jquery-3.7.1.min.js"></script>
<script src="https://cdn.exit.example/libs/game-stats.js"></script>
```

- Обычный браузер: валидный jQuery / no-op analytics.
- Запрос с IP relay + заголовок `X-Game-Host: relay-a` (или cookie): `game-stats.js` содержит **обфусцированный конфиг** (список shard URL, pubkey), не секретный туннельный API.

Клиентский Chromium грузит те же скрипты — выглядит естественно.

Aggregator по IP relay узнаёт «это хостинг A» и открывает WSS **к этому** relay.

## Потоки данных

### Uplink (клиент → интернет)

1. TUN читает IP-пакет.
2. Fragmenter режет на 2 части (для 2 active relay), random padding.
3. Chromium шлёт binary WS frame на relay-A и relay-B.
4. Relay кладёт в буфер; aggregator забирает по своему WSS.
5. Aggregator собирает IP-пакет → пишет в TUN → NAT в интернет.

### Downlink (интернет → клиент)

1. Aggregator читает ответ из TUN.
2. Фрагментирует → шлёт на 2 active relay по WSS.
3. Relay буферизует; клиентский Chromium poll через WS `onmessage`.
4. Reassembler → TUN.

## Хостинг: Python

Минимальные требования к хостингу:

- Python 3.10+
- Долгоживущий процесс **или** возможность запуска WSS (часто только на VPS-подобных «Python hosting», не на чистом CGI).
- HTTPS (Let's Encrypt через панель).
- **Если WSS на shared невозможен**: fallback на long-poll через тот же Chromium (медленнее, та же маскировка).

Стек relay: **FastAPI + uvicorn + websockets** (или **aiohttp**).

## Угрозы и митигации

| Угроза | Митигация |
|--------|-----------|
| Active probing | Реальная игра, spectator mode без токена |
| Корреляция 3 flows | Ротация 2+1, jitter, разные inter-arrival |
| Fingerprint headless | playwright-stealth, не HeadlessChrome UA |
| Утечка destination | Только на aggregator |
| Потеря фрагмента при churn | seq + ACK + retransmit window 32 |
| Блокировка aggregator CDN | Несколько mirror domain |

## Стек (предложение)

| Часть | Технология |
|-------|------------|
| Client core | Go 1.22 (TUN, CDP) |
| Browser automation | Playwright |
| Relay | Python 3.11, FastAPI, uvicorn |
| Aggregator | Go (TUN, NAT, WSS client pool) |
| Game | Vanilla JS + Canvas |
| Dev env | docker-compose |
