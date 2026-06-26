# 3wayproxy

R&D multipath-туннель: **Go client + aggregator (Linux)**, **Python relay**, binary WebSocket carrier.

## Структура

```
3wayproxy/
├── client/          # Go: TUN → relay WSS (/ws/play)
├── aggregator/      # Go: relay WSS (/ws/spectator) → TUN + NAT
├── relay/           # Python FastAPI: буфер между player и spectator
├── shared/          # Go: proto, fragment, reasm, tun (linux)
├── config/          # YAML конфиги
├── scripts/         # запуск на Ubuntu
└── bin/             # собранные бинарники (после make)
```

## Требования (Ubuntu 24)

- `golang-go`, `python3`, `python3-venv`
- `ip` (iproute2), `iptables` — для TUN и NAT
- **root/sudo** для client и aggregator (TUN + NAT)

## Быстрый старт

```bash
./scripts/setup-dev.sh

# Терминал 1 — relay
./scripts/run-relay.sh

# Терминал 2 — aggregator (sudo)
./scripts/run-aggregator.sh

# Терминал 3 — client
# ВАЖНО: aggregator и client на ОДНОЙ машине → client только через netns:
./scripts/run-client-netns.sh

# Терминал 4 — проверка (ping внутри netns клиента)
sudo ip netns exec 3wayclient ping -c 3 8.8.8.8
```
`session_id` в `config/client.dev.yaml` и `config/aggregator.dev.yaml` должен совпадать.

Маршруты в client по умолчанию: только `8.8.8.8` и `1.1.1.1` через TUN (не весь интернет).

## Сборка вручную

```bash
make build    # bin/3wayproxy-client, bin/3wayproxy-agg
make test     # unit-тесты shared/
make relay    # uvicorn на :8000
```

## Docker (только relay)

```bash
make relay-docker
# ws://127.0.0.1:8000/ws/play
```

## Деплой на удалённый сервер

См. [docs/DEPLOY.md](docs/DEPLOY.md).

```bash
./scripts/build-release.sh amd64
./scripts/check-binary.sh dist/3wayproxy-agg
```

- [ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [PLAN.md](docs/PLAN.md)
- [PROTOCOL.md](docs/PROTOCOL.md)

## Статус

| Компонент | Статус |
|-----------|--------|
| shared/proto + reasm | готово |
| relay WSS | фаза 1 |
| Go client TUN | фаза 1 |
| Go aggregator TUN+NAT | фаза 1 |
| 3 relay + ротация 2+1 | готово — [PHASE2.md](docs/PHASE2.md) |
| Chromium carrier (stub) | в работе — [PHASE3.md](docs/PHASE3.md) |
| Шахматы (active probing) | [RUNBOOK.md](docs/RUNBOOK.md) |
