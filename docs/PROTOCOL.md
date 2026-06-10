# Протокол 3wayproxy

Бинарный carrier поверх **WebSocket binary frames** (`opcode=2`). Текстовые WS-кадры используются только для cover (JSON score events).

## Версия

- `PROTO_VER = 1`

## Типы кадров (1 byte)

| Code | Name | Направление |
|------|------|-------------|
| `0x01` | HANDSHAKE | client → relay |
| `0x02` | HANDSHAKE_ACK | relay → client |
| `0x03` | RESUME | client → relay |
| `0x04` | FRAGMENT | client ↔ relay ↔ aggregator |
| `0x05` | ACK | both ways |
| `0x06` | HEARTBEAT | cover keepalive |
| `0x07` | COVER_EVENT | JSON в binary wrapper (опционально) |

## HANDSHAKE (client → relay)

```
offset  size  field
0       1     type = 0x01
1       1     proto_ver
2       2     shard_id (0..2)
4       8     session_id (uint64)
12      16    client_nonce
28      32    hmac_sha256(session_key_material, client_nonce)  // опционально фаза 3+
```

`session_id` генерируется клиентом один раз на TUN-сессию; сохраняется в `localStorage`.

## HANDSHAKE_ACK (relay → client)

```
0       1     type = 0x02
1       1     proto_ver
2       8     session_id
10      2     mtu_hint (default 1200)
12      1     relay_shard_id
```

## RESUME (после WSS churn)

```
0       1     type = 0x03
1       8     session_id
9       4     last_seq_sent (uint32)
13      4     last_seq_recv (uint32)
```

## FRAGMENT

```
0       1     type = 0x04
1       8     session_id
9       4     packet_id (uint32)      // IP-packet id
13      2     frag_idx (uint16)
15      2     frag_total (uint16)
17      2     payload_len (uint16)
19      N     payload
19+N    P     padding (random 0..64)  // P = random
```

- `packet_id` монотонный per session.
- `frag_total` обычно 2 при двух active relay (каждый relay получает свой `frag_idx`).
- CRC: **xxhash32** от `packet_id|frag_idx|payload` в отдельном ACK (не в каждом фрагменте).

## ACK

```
0       1     type = 0x05
1       8     session_id
9       4     packet_id
13      2     frag_idx
15      1     status (0=ok, 1=need_retx)
```

## HEARTBEAT / COVER

HEARTBEAT: 4 байта timestamp (uint32 unix).

COVER_EVENT: после HEARTBEAT идёт JSON `{"score":120,"dir":"left"}` — для pcap выглядит как game telemetry.

## Aggregator ↔ Relay (тот же binary, другой handshake)

Отдельный WS path: `/ws/spectator` (в коде — внутреннее имя, снаружи — «watch match»).

```
HANDSHAKE_AGG:
  type = 0xA1
  relay_id (1 byte)
  aggregator_token (32 bytes HMAC)
```

Relay проверяет token против config; не светит endpoint в публичном JS.

## Ротация 2+1 (control plane)

Клиент шлёт COVER_EVENT при смене relay:

```json
{"event":"player_pause","duration_ms":1000}
```

В carrier (не в JSON): служебный кадр не нужен — aggregator читает **ротацию из config**, синхронизированную по `session_id` и wall-clock, или клиент дублирует `ROTATE_HINT` frame (type `0x10`, фаза 2).

## Reassembly (aggregator и client)

1. Буфер: `map[packet_id] → fragments[]`, deadline = now + 5s.
2. Когда `frag_total` фрагментов собраны → concat → проверка xxhash32 (optional) → IP packet.
3. Неполный packet_id по TTL → drop, send ACK `need_retx`.

## MTU

- TUN MTU: **1200**
- Max fragment payload: **512** (с padding до ~576)
- Два фрагмента на пакет ≈ 1024 + headers < 1200

## Маскировка WS URL

Публичные пути (пример):

| URL | Роль |
|-----|------|
| `/ws/play` | клиент (игрок) |
| `/ws/spectator` | aggregator |
| `/api/score` | cover HTTP POST |

Subprotocol: `snake-game-v1` (выглядит как версия игры).
