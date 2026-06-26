# Деплой aggregator / client на удалённый Linux

## Ошибка `Syntax error: ")" unexpected`

Shell **пытается выполнить файл как скрипт**, а не как бинарник.

| Причина | Что делать |
|---------|------------|
| Запуск `sh ./bin/3wayproxy-agg` | Только `./bin/3wayproxy-agg` после `chmod +x` |
| На сервере не ELF (текст/битый файл) | `file ./bin/3wayproxy-agg` → должно быть `ELF 64-bit` |
| Архитектура не совпадает | `uname -m` на сервере vs `file` бинарника |
| `bin/` не копировали (в .gitignore) | Собрать на сервере или scp из `dist/` |

```bash
# на удалённом сервере
./scripts/check-binary.sh ./bin/3wayproxy-agg
```

---

## Не копируйте через scp

| Каталог | Почему |
|---------|--------|
| `relay/.venv/` | shebang `pip`/`python` указывает на интерпретатор **другой** машины → `cannot execute: required file not found` |
| `bin/` | лучше собрать на целевой архитектуре (`make build`) |
| `.tools/` | локальный Go toolchain |

На сервере после `scp` проекта: `rm -rf relay/.venv && ./scripts/setup-dev.sh`

---

## Рекомендуемый способ

### Вариант A — собрать на сервере (проще всего)

```bash
# на удалённом Ubuntu
sudo apt install golang-go git
git clone ... 3wayproxy && cd 3wayproxy
make build
sudo ./scripts/run-aggregator.sh
```

### Вариант B — собрать локально, скопировать

```bash
# на dev-машине — узнайте архитектуру сервера: ssh user@server uname -m
./scripts/build-release.sh amd64    # x86_64
# или
./scripts/build-release.sh arm64    # aarch64 / Raspberry Pi

scp dist/3wayproxy-agg-linux-amd64 user@server:~/3wayproxy/bin/3wayproxy-agg
scp config/aggregator.dev.yaml user@server:~/3wayproxy/config/
ssh user@server 'chmod +x ~/3wayproxy/bin/3wayproxy-agg && ~/3wayproxy/scripts/check-binary.sh ~/3wayproxy/bin/3wayproxy-agg'
```

---

## Конфиг для удалённого aggregator

`config/aggregator.dev.yaml` — поправьте relay:

```yaml
relay_ws: ws://192.168.88.26:8000/ws/spectator   # IP машины с relay, не 127.0.0.1
```

Relay должен слушать `0.0.0.0:8000`, не только localhost (для удалённого aggregator).

---

## Client и aggregator на разных машинах

- **Client** — ваш ПК (`run-client.sh` или `run-client-netns.sh` если relay локально)
- **Aggregator** — удалённый VPS (`run-aggregator.sh`)
- **Relay** — локально или на хостинге; WSS должен быть доступен с обеих сторон

На удалённом aggregator **не нужен** netns — конфликт TUN только когда client и aggregator на **одном** хосте.

---

## Статическая сборка

`scripts/build-release.sh` использует `CGO_ENABLED=0` — бинарник не зависит от glibc на сервере.

TUN всё равно требует Linux и `sudo`.
