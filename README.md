# 3wayproxy

R&D-проект мультипath-туннеля с маскировкой под браузерную игру (змейка / тетрис).

Трафик разбивается на фрагменты и передаётся через 2 из 3 relay-сайтов на shared-хостинге по WebSocket (binary). Соединения периодически рвутся и восстанавливаются — для DPI это похоже на переподключения игрока. Для клиента и aggregator сессия остаётся непрерывной.

## Компоненты

| Компонент | Роль |
|-----------|------|
| `client/` | TUN-интерфейс, headless Chromium, шардирование, ротация relay |
| `relay/` | Python: игра + WSS-сервер, буфер фрагментов |
| `aggregator/` | Exit: reassembly, TUN/NAT, CDN-библиотеки, WSS к relay |
| `game/` | Фронтенд игры (canvas, cover traffic) |
| `shared/` | Формат кадров, криптография, константы протокола |

## Документация

- [План разработки](docs/PLAN.md) — фазы, сроки, критерии готовности
- [Архитектура](docs/ARCHITECTURE.md) — потоки данных, ротация, WSS churn
- [Протокол](docs/PROTOCOL.md) — бинарный формат кадров и сессий

## Быстрый старт (после MVP-0)

```bash
# Локальная разработка — три relay + aggregator в docker-compose
docker compose -f deploy/docker-compose.dev.yml up

# Клиент (Linux, root для TUN)
sudo ./client/bin/3wayproxy-client --config config/client.dev.yaml
```

## Статус

| Фаза | Описание | Статус |
|------|----------|--------|
| 0 | Протокол + reassembly в памяти | planned |
| 1 | Один relay + aggregator, WSS | planned |
| 2 | Три relay + ротация 2+1 | planned |
| 3 | Headless Chromium + cover game | planned |
| 4 | Хостинг + маскировка CDN | planned |

## Лицензия

Только для личного R&D. Не для массового распространения.
