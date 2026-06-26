# HTTPS через туннель

TUN работает на уровне **IP**. HTTPS (TCP:443) проходит автоматически, если IP назначения направлен в `tun3way`.

## Быстрый тест: внешний IP через serv2

```bash
# 1. Узнать IP ifconfig.me
dig +short ifconfig.me A
# например: 34.160.111.145

# 2. Добавить в config/client.dev.3relay.yaml → tun.routes:
#    - 34.160.111.145
#    и bypass_routes (relay мимо TUN):
#    bypass_routes:
#      - 89.125.56.202

# 3. Пересобрать и запустить client
make build
sudo ./bin/3wayproxy-client --config config/client.dev.3relay.yaml

# 4. DNS через 8.8.8.8 (уже в routes) + запрос к ifconfig.me
curl -4 --dns-servers 8.8.8.8 https://ifconfig.me
# Должен показать IP serv2 (89.125.56.202), не ваш домашний IP
```

## Проверка маршрута

```bash
ip route get 34.160.111.145   # dev tun3way
ip route get 89.125.56.202    # dev enp4s0 (bypass)
```

## Логи aggregator при curl

```
tun write ... 10.0.0.2→34.160.111.145 proto=6   # TCP SYN
tun read  ... 34.160.111.145→10.0.0.2 proto=6   # ответ
```

## DNS

| Resolver | Что сделать |
|----------|-------------|
| `8.8.8.8` / `1.1.1.1` | уже в `routes` — UDP:53 идёт через туннель |
| Системный (провайдер) | DNS **мимо** туннеля; IP сайта добавляйте вручную в `routes` |

```bash
curl --dns-servers 8.8.8.8 https://example.com
```

## Несколько IP (CDN)

```bash
dig +short example.com A
# добавить все A-записи в routes
```

Или один запрос с фиксированным IP:

```bash
curl --resolve example.com:443:93.184.216.34 https://example.com/
```

## Весь трафик через туннель (осторожно)

```yaml
routes:
  - 0.0.0.0/1
  - 128.0.0.0/1
bypass_routes:
  - 89.125.56.202    # serv2 — обязательно!
```

Два половинных маршрута `/1` = default без перезаписи `0.0.0.0/0`.

## Пример конфига

`config/client.dev.https-example.yaml`
