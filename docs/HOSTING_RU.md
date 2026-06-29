# Российские хостинги: что можно разместить для 3wayproxy

Сводка по **reg.ru**, **Timeweb**, **Webnames**, **Beget** для relay (WebSocket + долгоживущий процесс + статика «шахматы»).

> Источники: официальные KB провайдеров (2024–2026), [Timeweb docs](https://timeweb.com/ru/docs/virtualnyj-hosting/prilozheniya-i-frejmvorki/ispolzovanie-node-js-i-npm/), [Beget Node.js KB](https://beget.com/ru/kb/how-to/web-apps/node-js), обзоры тарифов Webnames.

---

## Что нужно relay 3wayproxy

| Требование | Зачем |
|------------|--------|
| **Входящий HTTP/HTTPS** | Шахматы на `/`, статика |
| **WebSocket (WSS)** | `/ws/play`, `/ws/spectator` — binary tunnel |
| **Процесс 24/7** | Aggregator держит WS к relay |
| **Свой порт или nginx proxy** | Upgrade: websocket |
| **~128 MB RAM** | Node relay + chess |
| **Без исходящего «прокси»** | Relay только принимает |

---

## Сводная таблица

| Провайдер | Тип | Relay Python | Relay Node | WSS | Комментарий |
|-----------|-----|:------------:|:----------:|:---:|-------------|
| **reg.ru** | Виртуальный хостинг | ⚠️ | ❌ | ❌ | PHP/MySQL; Node/WS — нет |
| **reg.ru** | VPS/VDS | ✅ | ✅ | ✅ | Root, nginx, PM2 — **рекомендуется** |
| **Timeweb** | Виртуальный хостинг | ⚠️ | ❌* | ❌ | *Node только CLI, не как служба ([docs](https://timeweb.com/ru/docs/virtualnyj-hosting/prilozheniya-i-frejmvorki/ispolzovanie-node-js-i-npm/)) |
| **Timeweb** | VDS/VPS Cloud | ✅ | ✅ | ✅ | Root, PM2, nginx |
| **Webnames** | Виртуальный хостинг | ❌ | ❌ | ❌ | Только PHP; нет Node/Python на shared |
| **Webnames** | VPS (от ~500 ₽/мес) | ✅ | ✅ | ✅ | KVM, Debian/Ubuntu |
| **Beget** | Виртуальный хosting | ⚠️ | ⚠️ | ⚠️ | Node.js через панель ([KB](https://beget.com/ru/kb/how-to/web-apps/node-js)); WSS — настройка nginx |
| **Beget** | VPS | ✅ | ✅ | ✅ | PM2 + nginx из коробки marketplace |

**Легенда:** ✅ — подходит; ⚠️ — возможно с ограничениями; ❌ — не подходит для tunnel relay.

---

## По провайдерам

### Reg.ru

**Виртуальный хостинг** (~от 200 ₽/мес):
- PHP, MySQL, Python (ограниченно), cron.
- **Нельзя** запустить uvicorn/node как постоянный WS-сервер.
- **Можно** только зарегистрировать домен и направить A-запись на VPS.

**VPS/VDS** ([reg.ru/vps](https://www.reg.ru/vps/)):
- Root, любая ОС, nginx, SSL.
- **Python relay** или **relay-node** — полностью.
- Один VPS = один relay = один домен (shard).

### Timeweb

**Виртуальный хостинг**:
- Официально: *«Node.js можно как консольную утилиту; запуск в виде системной службы или веб-сервера **невозможен**»*.
- **WebSocket relay на shared — нет.**

**VDS/VPS** ([timeweb.com/cloud](https://timeweb.com/ru/services/vds/)):
- Root, cloud-init, API.
- **Лучший вариант Timeweb** для 3wayproxy.

### Webnames

**Виртуальный хosting** (Старт ~169 ₽/мес):
- PHP, nginx; **нет Node.js, Python, Ruby** на shared.
- Только статический сайт / PHP — **не relay**.

**VPS** (VPS-1 ~500 ₽/мес, 512 MB RAM):
- Debian/Ubuntu, KVM.
- **relay-node** предпочтительнее Python (меньше RAM).
- 512 MB — впритык; лучше VPS-2 (1 GB).

### Beget

**Виртуальный хостинг**:
- Есть инструкции по **Node.js на shared** (Docker/PM2 в панели).
- Теоретически можно поднять `relay-node` + прокси nginx.
- **Риски:** лимиты CPU/inodes, нет root, WSS нужно согласовать с поддержкой.
- **Для R&D probing** — попробовать; для tunnel — **надёжнее VPS**.

**VPS** ([beget.com/vps](https://beget.com/ru/vps)):
- Marketplace Node.js + PM2 + nginx.
- **Оптимальный вариант Beget** для relay.

---

## Что разместить где (практика)

| Компонент | Виртуальный хosting | VPS |
|-----------|---------------------|-----|
| Домен + DNS | ✅ все провайдеры | ✅ |
| Relay (tunnel) | ❌ почти везде | ✅ |
| Шахматы `/` | ⚠️ только если WS работает | ✅ |
| Aggregator (TUN+NAT) | ❌ | ✅ (один exit-VPS) |
| Client | ваш ПК | ваш ПК |

**Массовое «плодение» relay:**
1. **1 домен = 1 VPS (или 1 Beget VPS) = 1 relay-node instance**
2. Минимальный тариф (512 MB–1 GB) достаточен для Node relay.
3. Не три shard на одном IP — три **разных домена** на трёх хостах.
4. Автоматизация: zip `relay-node/` + `game/` → scp → `npm install` → `pm2 start`.

---

## Деплой relay-node на VPS (любой из провайдеров)

```bash
# на сервере (Ubuntu)
sudo apt update && sudo apt install -y nginx certbot python3-certbot-nginx
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
sudo npm install -g pm2

# файлы проекта в /opt/3wayproxy
cd /opt/3wayproxy/relay-node
npm install --omit=dev
RELAY_SHARD_ID=0 PORT=3000 pm2 start src/server.js --name relay
pm2 save && pm2 startup

# nginx + Let's Encrypt → wss://game.example.com
# фрагмент: deploy/relay-node/nginx-ws.conf
```

Проверка:

```bash
curl https://game.example.com/health
curl -I https://game.example.com/
# wss://game.example.com/ws/play — из client config
```

---

## Python relay vs relay-node на shared/VPS

| | Python (FastAPI) | Node (`relay-node/`) |
|--|------------------|----------------------|
| RAM | ~80–150 MB + venv | ~40–80 MB |
| Деплой | venv, uvicorn | `npm install`, `node` |
| Shared Beget | тяжело | проще (если Node разрешён) |
| VPS | ✅ | ✅ |
| Протокол | эталон | идентичный |

**Рекомендация:** для reg.ru / Timeweb / Webnames / Beget **VPS + relay-node + PM2 + nginx**.

---

## Чеклист перед добавлением хоста в client.yaml

```bash
curl -sf "https://YOUR_DOMAIN/health" | grep ok
curl -sf -o /dev/null -w '%{http_code}' "https://YOUR_DOMAIN/"
# WebSocket (нужен wscat или websocat):
# wscat -c wss://YOUR_DOMAIN/ws/chess
```

Скрипт: `./scripts/probe-relay-host.sh https://YOUR_DOMAIN` (если добавлен).

---

## Итог

| Вопрос | Ответ |
|--------|--------|
| Можно ли на **дешёвый виртуальный хosting**? | **Почти нет** (кроме эксперimentа Beget Node) |
| Можно ли на **VPS этих же провайдеров**? | **Да**, Python или **relay-node** |
| Что менять в проекте? | На VPS — **ничего**, только деплой; на shared — нужен **VPS или fallback long-poll** (не реализован) |
| Что использовать для массового деплоя? | **relay-node** на минимальных VPS + разные домены |

См. также: [RUNBOOK.md](RUNBOOK.md), [relay-node/README.md](../relay-node/README.md).
