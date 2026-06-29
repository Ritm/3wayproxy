# Деплой на Sprinthost (mrritm.xsph.ru)

Официально: [Node.js на Sprinthost](https://help.sprinthost.ru/howto/nodejs) — запуск через **Phusion Passenger** (Apache).

## ⚠️ Важно: WebSocket

[Документация Sprinthost](https://help.sprinthost.ru/howto/nodejs):

> Модуль Passenger **не позволяет использовать WebSocket**.

| Что | На shared Sprinthost |
|-----|---------------------|
| Шахматы `https://mrritm.xsph.ru/` | ✅ |
| `/health`, `/game/*` | ✅ |
| **Туннель** `/ws/play`, `/ws/spectator` | ❌ |
| Aggregator → relay | ❌ |

**Полный relay** (tunnel) — только на **Sprintbox VDS** ([sprinthost.ru](https://sprinthost.ru)) или другом VPS с `node src/server.js` + PM2.

На shared можно проверить **обложку** (шахматы, active probing). Для туннеля используйте serv2 или Sprintbox.

---

## Структура файлов на хостинге

Через SSH путь обычно:

```text
/home/ВАШ_ЛОГИН/domains/mrritm.xsph.ru/public_html/
```

Нужно **два каталога** (не только relay-node):

```text
public_html/
├── app.js              ← из relay-node/app.js
├── passenger-app.js
├── package.json
├── package-lock.json
├── src/                ← server.js (для VDS; на shared не используется)
└── ../game/            ← ОБЯЗАТЕЛЬНО рядом с public_html
    ├── chess/
    ├── carrier/
    └── stub/
```

Проще: положить `game` **внутрь** public_html:

```text
public_html/
├── app.js
├── passenger-app.js
├── package.json
├── node_modules/       ← после npm install
└── game/
    ├── chess/
    ├── carrier/
    └── stub/
```

Тогда в панели или перед запуском:

```bash
export GAME_ROOT=/home/LOGIN/domains/mrritm.xsph.ru/public_html/game
```

(замените `LOGIN` на логин аккаунта)

---

## Пошагово

### 1. Панель Sprinthost

1. **Сайты** → **Веб-серверы** → **Добавить сервер** (если нет) → выберите **Node.js 22** (или 20+).
2. Подключите сайт **mrritm.xsph.ru** к этому веб-серверу.
3. Убедитесь, что домен `mrritm.xsph.ru` привязан к аккаунту (технический `*.xsph.ru`).

### 2. SSH

```bash
ssh LOGIN@ssh.sprinthost.ru
# или хост из панели «SSH-доступ»

cd ~/domains/mrritm.xsph.ru/public_html
```

Загрузите недостающие файлы:
- `game/` (с ПК: `rsync -avz game/ LOGIN@ssh.sprinthost.ru:~/domains/mrritm.xsph.ru/public_html/game/`)
- `app.js`, `passenger-app.js`, `package.json` из `relay-node/`

### 3. npm install

```bash
cd ~/domains/mrritm.xsph.ru/public_html
npm22 install --omit=dev
# или: npm install --omit=dev  (если default node 14 — лучше npm22)
```

### 4. Файл `.htaccess` в `public_html`

Замените `LOGIN` и путь:

```apache
SetEnv GHOST_NODE_VERSION_CHECK false
PassengerStartupFile app.js
PassengerResolveSymlinksInDocumentRoot on
Require all granted
PassengerAppType node
PassengerAppRoot /home/LOGIN/domains/mrritm.xsph.ru/public_html
Options -MultiViews
PassengerStickySessions on
SetEnv GAME_ROOT /home/LOGIN/domains/mrritm.xsph.ru/public_html/game
```

`PassengerAppRoot` — **полный путь** к каталогу с `app.js`.

### 5. Перезапуск Passenger

```bash
mkdir -p tmp
touch tmp/restart.txt
```

При отладке (медленно для продакшена):

```bash
touch tmp/always_restart.txt
# после отладки: rm tmp/always_restart.txt
```

### 6. Проверка

```bash
curl -s https://mrritm.xsph.ru/health
# ok

curl -s -o /dev/null -w '%{http_code}\n' https://mrritm.xsph.ru/
# 200 — шахматы
```

В браузере: `https://mrritm.xsph.ru/`

---

## Если нужен полный tunnel relay

### Вариант A — Sprintbox VDS (тот же Sprinthost)

1. Заказать [Sprintbox](https://sprinthost.ru) (VDS).
2. Деплой как в [HOSTING_RU.md](HOSTING_RU.md): `node src/server.js`, PM2, nginx, SSL.
3. В client.yaml: `wss://ваш-домен/ws/play`.

### Вариант B — оставить tunnel на serv2

- Relay **обложка** (шахматы): `mrritm.xsph.ru` на Sprinthost shared.
- Relay **tunnel**: serv2 `:81–83` или другой VPS.
- В фазе 4 — разные домены для 3 shard.

---

## Типичные ошибки

| Симптом | Решение |
|---------|---------|
| 500 / Passenger error | `touch tmp/restart.txt`; смотреть логи в панели |
| Нет стилей шахмат | Загрузить `game/`, проверить `GAME_ROOT` |
| `Cannot find module` | `npm22 install` в `public_html` |
| WS не подключается | **Ожидаемо** на shared — нужен VDS |
| Node 14 по умолчанию | Использовать веб-сервер **Node 22** в панели |

---

## Минимальный набор файлов для загрузки (shared)

С ПК:

```bash
LOCAL=/home/user/ownCloud/3wayproxy
HOST=LOGIN@ssh.sprinthost.ru
REMOTE=~/domains/mrritm.xsph.ru/public_html

rsync -avz "$LOCAL/relay-node/app.js" \
           "$LOCAL/relay-node/passenger-app.js" \
           "$LOCAL/relay-node/package.json" \
           "$LOCAL/relay-node/package-lock.json" \
           "$HOST:$REMOTE/"

rsync -avz "$LOCAL/game/" "$HOST:$REMOTE/game/"
```

После этого — SSH, `npm22 install`, `.htaccess`, `touch tmp/restart.txt`.
