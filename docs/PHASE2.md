# Фаза 2 — 3 relay + ротация 2+1

## Что сделано

- `shared/rotate` — синхронная схема 2 active + 1 idle
- `shared/pool` — 3 WebSocket, шардирование фрагментов, churn + RESUME
- Client и aggregator: **обратная совместимость** с 1 relay (`relay_ws` в конфиге)
- Relay: обработка `RESUME` (0x03)

## Запуск (локально, 3 relay)

```bash
make build

# Терминал 1 — три relay
./scripts/run-relays.sh

# Терминал 2 — aggregator (удалённый или локальный)
sudo ./bin/3wayproxy-agg --config config/aggregator.dev.3relay.yaml

# Терминал 3 — client
sudo ./bin/3wayproxy-client --config config/client.dev.3relay.yaml
# или netns, если aggregator на том же хосте:
# sudo ip netns exec 3wayclient ./bin/3wayproxy-client --config config/client.dev.3relay.yaml

# Терминал 4
ping -c 5 8.8.8.8
```

## Ваша схема (client локально, aggregator удалённо)

1. **Relay** на локальном ПК: `./scripts/run-relays.sh` (слушает `0.0.0.0:8001-8003`)
2. В `client.dev.3relay.yaml` — `127.0.0.1` для ws
3. В `aggregator.dev.3relay.yaml` на сервере — **LAN/public IP** вашего ПК:
   ```yaml
   relays:
     - id: 0
       ws: ws://192.168.88.26:8001/ws/spectator
     ...
   ```
4. `session.rotate_interval_sec` и `session_id` **одинаковые** на client и aggregator

## Логи фазы 2

```
pool: rotate epoch=… idle=2 active=0,1
pool: churn relay 1
```

## Docker

```bash
make relay-docker
# ws://127.0.0.1:8001|8002|8003/ws/play
```

## Следующий шаг — фаза 3

Headless Chromium + змейка (carrier только через браузер).
