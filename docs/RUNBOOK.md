# Пошаговый runbook — 3wayproxy

Полная инструкция: сборка, запуск relay / aggregator / client, сбор pcap и анализ.

**Ваш деплой:**
- Сервер: `serv2.erofeevonline.ru` (`/data/3wayproxy`)
- Relay: порты **81, 82, 83** (без nginx)
- Client: ваш ПК (Linux)
- Aggregator: на serv2 рядом с relay

---

## 0. Подготовка (один раз)

### На сервере (`/data/3wayproxy`)

```bash
sudo apt install -y golang-go python3 python3-venv tcpdump
cd /data/3wayproxy
rm -rf relay/.venv   # если копировали с другой машины
./scripts/setup-dev.sh
pip install chess   # или: relay/.venv/bin/pip install -r relay/requirements.txt
make build
chmod +x scripts/*.sh
```

### На клиенте (ПК)

```bash
cd ~/ownCloud/3wayproxy   # или ваш путь
git checkout phase-3
make build
./scripts/install-chromium.sh   # для carrier: browser
```

Конфиги (не в git): `config/client.dev.3relay.yaml`, `config/aggregator.dev.3relay.yaml` — `session_id` **должен совпадать**.

---

## 1. Запуск relay (сервер)

```bash
ssh root@serv2.erofeevonline.ru
cd /data/3wayproxy

# остановить старые
pkill -f 'uvicorn app.main:app' || true

# запуск (foreground — для отладки)
./scripts/run-relays-81.sh
```

Фон:

```bash
nohup ./scripts/run-relays-81.sh >> /var/log/3wayproxy-relays.log 2>&1 &
```

**Проверка:**

```bash
curl -s http://127.0.0.1:81/health          # ok
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:81/        # 200 — шахматы
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:81/play.html
ss -tlnp | grep -E ':81|:82|:83'
```

В браузере: `http://serv2.erofeevonline.ru:81/` — анонимные шахматы (active probing).

---

## 2. Запуск aggregator (сервер)

Отдельный терминал / screen на serv2:

```bash
cd /data/3wayproxy
sudo ./bin/3wayproxy-agg --config config/aggregator.dev.3relay.yaml
```

В конфиге aggregator relay должны быть **localhost:81–83**:

```yaml
relays:
  - ws: ws://127.0.0.1:81/ws/spectator
  - ws: ws://127.0.0.1:82/ws/spectator
  - ws: ws://127.0.0.1:83/ws/spectator
```

**Проверка:** в логах `pool: connected relay …`

---

## 3. Запуск client (ваш ПК)

### Native WebSocket (фаза 2)

```bash
cd /path/to/3wayproxy
sudo ./bin/3wayproxy-client --config config/client.dev.3relay.yaml
```

### Chromium carrier (фаза 3)

В конфиге:

```yaml
carrier: browser
browser:
  headless: true
```

```bash
sudo ./bin/3wayproxy-client --config config/client.dev.3relay.yaml
```

**Проверка туннеля:**

```bash
ping -c 3 8.8.8.8
curl --dns-servers 8.8.8.8 https://ifconfig.me
```

> Client и aggregator на **одной** машине без netns — ping не работает. У вас client на ПК, aggregator на serv2 — netns не нужен.

---

## 4. Сбор данных на сервере

Во время работы туннеля (5–30 мин, желательно с HTTPS/сёрфингом):

```bash
cd /data/3wayproxy
./scripts/capture-relay.sh /tmp/capture-relay.pcap
# Ctrl+C после теста
```

Или вручную:

```bash
sudo tcpdump -ni any 'tcp port 81 or tcp port 82 or tcp port 83' -w /tmp/capture-relay.pcap
```

Скопировать pcap на ПК (опционально):

```bash
scp root@serv2.erofeevonline.ru:/tmp/capture-relay.pcap .
```

---

## 5. Сбор данных на клиенте

```bash
cd /path/to/3wayproxy
./scripts/capture-client.sh serv2.erofeevonline.ru capture-client.pcap
# Ctrl+C
```

Или:

```bash
sudo tcpdump -ni any host serv2.erofeevonline.ru -w capture-client.pcap
```

Запускайте **до** старта client и останавливайте после нагрузки.

---

## 6. Анализ pcap

На машине с `tshark`:

```bash
sudo apt install tshark   # один раз

./scripts/pcap-analysis.sh capture-client.pcap --host serv2.erofeevonline.ru
./scripts/pcap-analysis.sh capture-relay.pcap --ports 81,82,83
```

Смотрите:
- **Корреляция |r|** между ногами 81/82/83 (ниже — лучше для маскировки)
- **Длительность TCP-потоков** (много >60s — похоже на VPN)
- **JA3** (при `carrier: browser` — должен быть как Chrome)
- **Синхронные старты** трёх ног

Сравните с эталоном: pcap обычного браузера на `http://serv2:81/` (шахматы, без туннеля).

---

## 7. Обновление файлов на сервере (без git)

С вашего ПК:

```bash
LOCAL=/home/user/ownCloud/3wayproxy
SERVER=root@serv2.erofeevonline.ru
REMOTE=/data/3wayproxy

rsync -avz "$LOCAL/relay/" "$SERVER:$REMOTE/relay/"
rsync -avz "$LOCAL/game/" "$SERVER:$REMOTE/game/"
```

На сервере:

```bash
cd /data/3wayproxy/relay && source .venv/bin/activate
pip install -r requirements.txt
pkill -f uvicorn; sleep 1
cd /data/3wayproxy && nohup ./scripts/run-relays-81.sh >> /var/log/3wayproxy-relays.log 2>&1 &
```

Aggregator пересобирать только если менялся Go-код aggregator.

---

## 8. Active probing (шахматы)

| URL | Назначение |
|-----|------------|
| `http://serv2:81/` | Шахматы — для цензора / бота |
| `http://serv2:81/play.html` | Carrier (Chromium client) |
| `ws://serv2:81/ws/play` | Туннель (binary) |
| `ws://serv2:81/ws/chess` | Игра (JSON) |

Сценарий для probing:
1. Открыть `/` — видна игра
2. «Играть белыми» → ожидание
3. Второй браузер → «Присоединиться»
4. Партия без регистрации

---

## 9. Типичные проблемы

| Симптом | Решение |
|---------|---------|
| `play.html` 404 | Обновить `relay/app/main.py` + `game/` |
| Aggregator не коннектится | Порты 81–83, не 8001; relay слушает `0.0.0.0` |
| ping не идёт | aggregator запущен? `session_id` совпадает? |
| Chromium не стартует | `./scripts/install-chromium.sh` |
| chess ImportError | `pip install chess` в relay venv |

---

## Порядок запуска (кратко)

```text
1. relay (serv2)     ./scripts/run-relays-81.sh
2. aggregator (serv2) sudo ./bin/3wayproxy-agg --config ...
3. [опционально] tcpdump на serv2 и на ПК
4. client (ПК)       sudo ./bin/3wayproxy-client --config ...
5. тест ping/curl
6. Ctrl+C tcpdump → pcap-analysis.sh
```
