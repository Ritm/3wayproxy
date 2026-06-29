# Relay на Node.js

Альтернатива Python/FastAPI relay. Тот же протокол (`shared/proto`), те же endpoint'ы.

## Быстрый старт

```bash
cd relay-node
npm install
RELAY_SHARD_ID=0 PORT=8001 node src/server.js
```

Или из корня:

```bash
./scripts/run-relay-node.sh          # PORT=8000
./scripts/run-relays-node-81.sh      # 81, 82, 83
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `PORT` | `8000` | HTTP + WebSocket |
| `RELAY_SHARD_ID` | `0` | shard для HANDSHAKE_ACK |

## Endpoint'ы

| Путь | Назначение |
|------|------------|
| `/` | Шахматы (active probing) |
| `/play.html` | Carrier page (Chromium) |
| `/game/*` | Статика (chess, carrier) |
| `/health` | `ok` |
| `/ws/play` | Binary tunnel (client) |
| `/ws/spectator` | Binary tunnel (aggregator) |
| `/ws/chess` | JSON (игра) |

## Деплой на VPS (PM2)

```bash
cd relay-node && npm install --omit=dev
pm2 start ../deploy/relay-node/ecosystem.config.cjs
pm2 save
```

Nginx: см. `deploy/relay-node/nginx-ws.conf`.

## Зависимости

- Node.js **≥ 18**
- `express`, `ws`, `chess.js`
- Каталог `../game/` (общий с Python relay)

Client и aggregator **не меняются** — те же URL `wss://domain/ws/play`.
