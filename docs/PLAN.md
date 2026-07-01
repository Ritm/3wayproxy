# План разработки 3wayproxy

Личный R&D. Ориентир: работающий ping/curl через туннель с маскировкой под игру, 3 relay, ротация 2+1, WSS churn.

---

## Фаза 0 — Протокол и reassembly (локально, без браузера)

**Цель**: доказать, что фрагментация IP-пакетов и сборка работают.

**Задачи**

- `shared/proto/` — кодирование/декодирование кадров (Go)
- `aggregator/reassembly/` — unit-тесты: 2 фрагмента → 1 IP packet
- `client/fragment/` — нарезка TUN read на 2 FRAGMENT кадра
- Mock-тест: client mock → aggregator mock по TCP (вместо WSS)

**Критерий готовности**

```bash
go test ./...
# ping 8.8.8.8 через mock-туннель — 0% loss в локали
```

**Срок**: 3–5 дней

---

## Фаза 1 — Один relay + aggregator (WSS)

**Цель**: end-to-end через один WebSocket без игры.

**Задачи**

- `relay/` — FastAPI app:
  - `GET /` — заглушка игры
  - `WS /ws/play` — binary frames
  - `WS /ws/spectator` — aggregator
  - in-memory buffer `session_id → queue[fragments]`
- `aggregator/` — WSS client к `/ws/spectator`, TUN interface
- `client/` — простой WSS client (временно **без** Chromium) к `/ws/play`
- `deploy/docker-compose.dev.yml` — relay + aggregator

**Критерий готовности**

- curl `https://ifconfig.me` через TUN с одним relay
- latency < 500 ms на пустом канале (локально)

**Срок**: 5–7 дней

**Риск**: хостинг без WSS — на dev используем docker; для prod проверить хостинг на фазе 4.

---

## Фаза 2 — Три relay + ротация 2+1 ✅

**Цель**: шардирование и смена active/idle relay. См. [PHASE2.md](PHASE2.md).

**Задачи**

- `shared/rotate/` + `shared/pool/` — 2 active, 1 idle
- client + aggregator: 3 WSS, шардирование фрагментов
- WSS churn + RESUME
- Retransmit: ACK `need_retx` (отложено)

**Критерий готовности**

- 3 контейнера relay в compose
- ping стабилен при ротации и churn
- в логах relay видны только binary WS, не целые HTTP к blocked hosts

**Срок**: 7–10 дней

---

## Фаза 3 — Headless Chromium + игра (cover)

**Цель**: carrier только через браузер; выглядит как игра.

**Задачи**

- `game/` — змейка на canvas (минимум: движение, score, game over)
- `game/carrier/ws_carrier.js` — bridge: `sendFragment(ArrayBuffer)`, `onFragment(cb)`
- `client/browser/` — Playwright:
  - 3 browser context (или tabs) на 3 relay URL
  - `exposeBinding('tunWrite', ...)` для downlink
  - autoplay snake между fragment sends (random 200–1000 ms)
- Убрать прямой WSS из native client (только CDP)
- playwright-stealth / аргументы против headless-detect

**Критерий готовности**

- Wireshark на стороне клиента: TLS Client Hello как Chrome; WS payload binary
- Ручной opening relay URL в браузере — игра работает без клиента
- curl через туннель с Chromium carrier

**Срок**: 10–14 дней

---

## Фаза 4 — Деплой на shared + CDN aggregator

**Цель**: реальные 3 домена на хостинге + exit-сервер.

**Задачи**

- Выбор 3 хостингов с Python и **долгоживущим** процессом (проверка WSS)
- `relay/deploy/` — systemd/supervisor, nginx reverse proxy → uvicorn
- HTTPS на каждом relay
- `aggregator/cdn/` — jquery + game-stats.js с IP-aware config
- Встраивание `<script src="cdn.../game-stats.js">` в страницу игры
- Aggregator token для `/ws/spectator` — env secret per relay
- Документ `docs/HOSTING_CHECKLIST.md` по результатам проверки

**Критерий готовности**

- Три реальных домена; active probing — играется змейка
- Личный сёрфинг через туннель 30 мин без 429
- Aggregator WSS до каждого relay стабилен 24h

**Срок**: 7–14 дней (зависит от хостинга)

**Fallback**: если WSS на shared невозможен — `docs/PLAN-FALLBACK-LONGPOLL.md` (создать при необходимости).

---

## Фаза 6 — Android (отдельно, после desktop MVP)

**Контекст**: детекция VPN актуальна **только на Android**. Linux/macOS/Windows ведутся с TUN без этой оговорки.

**Цель**: carrier через «игру» без обязательного системного TUN.

**Задачи**

- `client/android/` — browser-only: WebView + та же `game/carrier`
- Опционально: `VpnService` + `addAllowedApplication()` + policy (app + domain)
- Документ `docs/ANDROID.md` — tradeoffs, детект, per-app routing

**Критерий готовности**

- Браузер на Android открывает t.me через туннель; иконка VPN **не требуется** (browser-only режим)
- Или: per-app VPN только для выбранных packages

**Срок**: после фазы 4, по необходимости

---

## Фаза 5 — Hardening (опционально)

- Padding buckets (256/512/768)
- Decoy POST `/api/score` без fragment
- Шифрование payload fragment (AES-GCM, ключ из HANDSHAKE)
- WebRTC datachannel client↔aggregator как второй путь
- Метрики: loss, reassembly latency, churn count

**Срок**: по мере необходимости

---

## Структура репозитория (целевая)

```
3wayproxy/
├── README.md
├── docs/
│   ├── PLAN.md
│   ├── ARCHITECTURE.md
│   └── PROTOCOL.md
├── shared/
│   └── proto/              # Go: frame codec
├── client/
│   ├── cmd/3wayproxy-client/
│   ├── tun/
│   ├── fragment/
│   ├── rotate/
│   └── browser/            # Playwright
├── relay/
│   ├── app/                # FastAPI
│   ├── ws/
│   ├── buffer/
│   └── requirements.txt
├── aggregator/
│   ├── cmd/3wayproxy-agg/
│   ├── tun/
│   ├── wspool/
│   └── cdn/static/
├── game/
│   ├── snake/
│   └── carrier/
├── deploy/
│   ├── docker-compose.dev.yml
│   └── nginx/
└── config/
    ├── client.example.yaml
    └── aggregator.example.yaml
```

---

## Конфиг (черновик)

### client.yaml

```yaml
session:
  churn_interval_sec: [2, 5]   # random range
  rotate_interval_sec: [3, 8]
  idle_pause_ms: 1000

relays:
  - url: https://game-a.example/
    ws: wss://game-a.example/ws/play
    shard_id: 0
  - url: https://game-b.example/
    ws: wss://game-b.example/ws/play
    shard_id: 1
  - url: https://game-c.example/
    ws: wss://game-c.example/ws/play
    shard_id: 2

browser:
  profile_dir: ~/.3wayproxy/chrome
  headless: true
  stealth: true

tun:
  name: tun0
  mtu: 1200
```

### aggregator.yaml

```yaml
relays:
  - id: 0
    spectator_ws: wss://game-a.example/ws/spectator
    token_env: RELAY_A_TOKEN
  - id: 1
    spectator_ws: wss://game-b.example/ws/spectator
    token_env: RELAY_B_TOKEN
  - id: 2
    spectator_ws: wss://game-c.example/ws/spectator
    token_env: RELAY_C_TOKEN

cdn:
  listen: ":443"
  paths:
    - /libs/jquery-3.7.1.min.js
    - /libs/game-stats.js

tun:
  name: tun1
  nat: true
```

---

## Порядок работ (что делать сейчас)

1. **Инициализировать Go module** в `shared/`, `client/`, `aggregator/`
2. **Реализовать codec** по `PROTOCOL.md` + тесты
3. **Mock e2e** (фаза 0)
4. **FastAPI relay** с двумя WS endpoint (фаза 1)
5. Параллельно: **минимальная змейка** в `game/` (можно с фазы 3, но HTML уже на фазе 1)

---

## Открытые вопросы (решить в процессе)


| #   | Вопрос                                         | Когда                                      |
| --- | ---------------------------------------------- | ------------------------------------------ |
| 1   | Go vs Rust для client                          | фаза 0 (предлагается Go: TUN + Playwright) |
| 2   | Поддерживает ли выбранный хостинг uvicorn WSS  | фаза 4                                     |
| 3   | Один Chromium profile / три tab vs три context | фаза 3                                     |
| 4   | Нужен ли SQLite buffer на relay при churn      | фаза 2                                     |
| 5   | Android: browser-only vs per-app VPN           | фаза 6                                     |


---

## Метрики успеха проекта

- Active probing: игра запускается, нет явного «proxy API»
- Ни один relay не логирует destination IP
- WSS сессии < 5 s в среднем (churn)
- Ротация 2+1 без потери > 1% пакетов на личном трафике
- Client TLS выглядит как Chrome (JA3 сравнение вручную)

