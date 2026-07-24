# Деплой Python relay на Amvera

[Документация Amvera: Python pip](https://docs.amvera.ru/applications/environments/python-pip.html)

Amvera поддерживает **FastAPI + uvicorn + WebSocket** (`wss://`). Для туннеля используйте `wss://`, не `ws://`.

---

## Что загрузить в «Code» (корень репозитория Amvera)

**Самый простой способ:** на ПК выполнить `bash deploy/amvera/pack.sh` и загрузить всё из `deploy/amvera-upload/` в корень Amvera.

Или скопируйте готовую папку `**deploy/amvera/`** целиком (там уже есть `app/`, `game/`, `run.py`).

Структура **плоская** — без каталога `relay/`:

```text
корень проекта на Amvera/
├── amvera.yml          # или amvera.yaml
├── requirements.txt
├── run.py              # точка входа (рекомендуется)
├── app/
│   ├── __init__.py
│   ├── main.py
│   ├── protocol.py
│   └── chess.py
└── game/
    ├── chess/
    ├── carrier/
    └── stub/
```

### Откуда копировать (локально)


| На Amvera          | Из репозитория 3wayproxy         |
| ------------------ | -------------------------------- |
| `app/`             | `relay/app/`                     |
| `game/`            | `game/`                          |
| `run.py`           | `deploy/amvera/run.py`           |
| `requirements.txt` | `deploy/amvera/requirements.txt` |
| `amvera.yml`       | `deploy/amvera/amvera.yml`       |


**Не загружайте:** `relay/.venv/`, `relay-node/`, `bin/`, `.git`, скрипты `.sh`.

---

## Параметры запуска в панели Amvera

### Вариант 1 — `scriptName: run.py` (рекомендуется)

Amvera запускает `python3 run.py`. Uvicorn вызывается из Python-модуля — **не нужен** бинарник `uvicorn` в PATH.

Файл `amvera.yml` в корне:

```yaml
meta:
  environment: python
  toolchain:
    name: pip
    version: "3.11"

build:
  requirementsPath: requirements.txt
  useCache: false

run:
  scriptName: run.py
  containerPort: 80
```

В интерфейсе:


| Поле              | Значение           |
| ----------------- | ------------------ |
| **scriptName**    | `run.py`           |
| **command**       | *(пусто)*          |
| **requirements**  | `requirements.txt` |
| **containerPort** | `80`               |


### Вариант 2 — `command` с `python3 -m`

Если хотите command вместо scriptName:

```
python3 -m uvicorn app.main:app --host 0.0.0.0 --port 80
```

**Не используйте** просто `uvicorn ...` — на Amvera бинарник часто не в PATH.


| Поле              | Значение                                                   |
| ----------------- | ---------------------------------------------------------- |
| **scriptName**    | *(пусто)*                                                  |
| **command**       | `python3 -m uvicorn app.main:app --host 0.0.0.0 --port 80` |
| **containerPort** | `80`                                                       |


`scriptName` и `command` **взаимоисключающие**.

После изменения конфига нажмите **«Собрать»** (не только «Перезапустить»).

---

## Переменные окружения (опционально)

В разделе «Переменные» / Environment Amvera:


| Переменная       | Пример       | Зачем                                                               |
| ---------------- | ------------ | ------------------------------------------------------------------- |
| `RELAY_SHARD_ID` | `0`          | Номер shard в HANDSHAKE_ACK (для 3 relay — три приложения: 0, 1, 2) |
| `GAME_ROOT`      | `/путь/game` | Обычно не нужен, если `game/` лежит рядом с `app/`                  |


---

## Проверка после деплоя

Подставьте URL из панели Amvera (`https://ваш-проект.amvera.io`):

```bash
curl -s https://ваш-проект.amvera.io/health
# ok

curl -s -o /dev/null -w '%{http_code}\n' https://ваш-проект.amvera.io/
# 200 — шахматы
```

В браузере: `https://ваш-проект.amvera.io/`

---

## Подключение client / aggregator

Один инстанс Amvera = **один relay** (один shard).

**Client** (`config/client...yaml`):

```yaml
relays:
  - ws: wss://ваш-проект.amvera.io/ws/play
    shard_id: 0
```

**Aggregator** (если aggregator на другом сервере):

```yaml
relays:
  - ws: wss://ваш-проект.amvera.io/ws/spectator
```

Три shard — три приложения Amvera (или три домена) с `RELAY_SHARD_ID=0`, `1`, `2`.

`session_id` на client и aggregator должен совпадать.

---

## Типичные ошибки


| Симптом                        | Решение                                                                                                                                                                                                                                                          |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `**No module named 'app'**`    | На сервере **нет папки `app/`** в корне. Частая ошибка: загрузили только `run.py` + `requirements.txt`, или оставили путь `relay/app/` вместо `app/`. Загрузите `deploy/amvera/` целиком. В корне Amvera должен быть файл `app/main.py`, не `relay/app/main.py`. |
| `ModuleNotFoundError: uvicorn` | Зависимости не установились — проверьте лог сборки и наличие `requirements.txt` в корне                                                                                                                                                                          |
| Нет шахмат / 404 на `/game`    | Загрузите каталог `game/` в корень                                                                                                                                                                                                                               |
| Сборка без зависимостей        | `requirements.txt` в корне, путь в `build.requirementsPath`                                                                                                                                                                                                      |
| WS не коннектится              | URL `wss://`, не `ws://`                                                                                                                                                                                                                                         |
| Приложение не стартует         | Проверить логи; порт в command = containerPort                                                                                                                                                                                                                   |


---

## Минимальный rsync с ПК (пример)

```bash
LOCAL=/home/user/ownCloud/3wayproxy
DEST=./amvera-upload   # затем залить в Amvera через git/веб

mkdir -p "$DEST/app" "$DEST/game"
cp -r "$LOCAL/relay/app/"* "$DEST/app/"
cp -r "$LOCAL/game/"* "$DEST/game/"
cp "$LOCAL/deploy/amvera/run.py" "$LOCAL/deploy/amvera/amvera.yml" "$LOCAL/deploy/amvera/requirements.txt" "$DEST/"
```

Загрузите содержимое `amvera-upload/` в репозиторий Amvera.

---

## Деплой из GitHub (monorepo)

Репозиторий **3wayproxy** — monorepo. На Amvera нужен **только relay**, не весь проект.

**Не делайте** `git push amvera main:master` из корня monorepo — на Amvera попадут `client/`, `aggregator/`, `docs/` и т.д. Docker-сборка из корня (`deploy/Dockerfile`) всё равно копирует только relay, но репозиторий на сервере будет лишним и тяжёлым.

### Правильный push на Amvera (relay-only)

```bash
bash deploy/amvera/push.sh https://git.amvera.ru/ritm/rel3
# или, если remote уже есть:
bash deploy/amvera/push.sh amvera-relay0
```

Скрипт собирает плоскую структуру (через `pack.sh`) и **force-push** в `master` на Amvera. В корне Amvera остаётся только:

```text
amvera.yml, requirements.txt, run.py, app/, game/
```

Повторите для каждого из трёх приложений (свой URL).

### Шаги (один shard = одно приложение Amvera)

1. [Amvera](https://amvera.ru) → **Создать проект** → git Amvera (URL на вкладке «Репозиторий»).
2. Локально: `bash deploy/amvera/push.sh <URL проекта>`.
3. **Переменные окружения** (для каждого из 3 приложений свой shard):

  | Приложение | `RELAY_SHARD_ID` |
  | ---------- | ---------------- |
  | relay-ritm | `0`              |
  | rel2-ritm  | `1`              |
  | rel3-ritm  | `2`              |

4. Дождаться **«Успешно развернуто»** на вкладке сборки.
5. Привязать домен (или использовать `*.amvera.io`).

### Обновление relay после изменений в monorepo

```bash
git push origin main                       # GitHub
bash deploy/amvera/push.sh amvera-relay0   # каждый shard отдельно
```

GitHub webhook на monorepo для Amvera **не обязателен**, если используете `push.sh`.

### Три relay — три проекта

Один репозиторий GitHub, **три отдельных приложения** Amvera с разными доменами и `RELAY_SHARD_ID=0/1/2`.

### Альтернатива без git push

```bash
bash deploy/amvera/pack.sh
# загрузить deploy/amvera-upload/ через веб-интерфейс Amvera → Code
```

Или git remote Amvera вручную (см. [документация Amvera](https://amvera.ru/doc_for_git)) — но только содержимое `amvera-upload/`, не monorepo.