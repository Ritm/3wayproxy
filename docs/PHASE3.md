# Фаза 3 — Chromium carrier (без игры)

**Цель:** WebSocket-трафик идёт только через headless Chromium (TLS/JA3 как Chrome).  
Игра-змейка — позже; сейчас stub-страница `/play.html`.

## Что сделано

- `game/carrier/ws_carrier.js` — handshake, fragment, resume в браузере
- `game/stub/play.html` — минимальная страница-обложка
- Relay отдаёт `/play.html` и `/game/*`
- `client/browser/` — Playwright, 3 browser context (по одному на relay)
- `carrier: browser` в конфиге клиента; aggregator остаётся на native WS

## Установка Chromium

```bash
./scripts/install-chromium.sh
```

Требуется: `go`, зависимости Playwright (скачивает Chromium ~150MB).

## Конфиг клиента

```yaml
carrier: browser

browser:
  headless: true
  profile_dir: ~/.3wayproxy/chrome   # опционально

session_id: 9000012345678901
relays:
  - ws: ws://127.0.0.1:8001/ws/play
    shard_id: 0
  # ...
```

`carrier: native` (по умолчанию) — прежний Go `gorilla/websocket`.

## Запуск (dev)

```bash
./scripts/setup-dev.sh
./scripts/run-relays.sh          # или один relay
./scripts/run-aggregator.sh      # sudo
sudo ./bin/3wayproxy-client -config config/client.dev.browser.yaml
```

Пример конфига: скопируйте `config/client.example.yaml` и задайте `carrier: browser`.

## Проверка

1. Откройте в обычном браузере: `http://127.0.0.1:8001/play.html?ws=ws://127.0.0.1:8001/ws/play&shard=0&session=9000012345678901` — статус «Carrier ready».
2. Клиент с `carrier: browser` — ping через TUN как в фазе 2.
3. Wireshark: TLS Client Hello от Chromium, не от Go.

## Дальше (в этой фазе)

- [x] Шахматы на `/` — анонимное лобби (active probing), см. [RUNBOOK.md](RUNBOOK.md)
- [ ] playwright-stealth / меньше headless-отпечатков
- [ ] Decoy fetch `/api/score` при churn

## Архитектура

```
TUN → Go fragmenter → Playwright Evaluate(sendBase64)
                              ↓
                    ws_carrier.js → WebSocket → relay
                              ↑
                    ExposeBinding(tunDeliver) → Go → TUN
```

Aggregator **не** использует Chromium — только клиент.
