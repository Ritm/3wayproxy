#!/usr/bin/env bash
# Анализ pcap для оценки «похожести» трафика 3wayproxy на обычную игру.
# Смотрит: длительность TCP/WS, bytes/s по ногам relay, корреляцию 3 потоков,
# распределение размеров сегментов, TLS ClientHello (JA3).
#
# Захват (клиент, во время работы туннеля):
#   sudo tcpdump -ni any host serv2.erofeevonline.ru -w capture-client.pcap
# Захват (serv2, relay):
#   sudo tcpdump -ni any 'port 81 or port 82 or port 83' -w capture-relay.pcap
#
# Использование:
#   ./scripts/pcap-analysis.sh capture.pcap
#   ./scripts/pcap-analysis.sh capture.pcap --ports 81,82,83 --bucket 1
#   ./scripts/pcap-analysis.sh capture.pcap --host 89.125.56.202
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PCAP=""
HOST_FILTER=""
PORTS="81,82,83"
BUCKET_SEC=1

usage() {
  sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
  echo
  echo "Опции:"
  echo "  --host HOST     фильтр ip (src или dst), напр. serv2 / 89.125.56.202"
  echo "  --ports LIST    порты relay через запятую (по умолчанию 81,82,83)"
  echo "  --bucket SEC    размер временного окна для bytes/s и корреляции (по умолчанию 1)"
  echo "  -h, --help      эта справка"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --host) HOST_FILTER="${2:?}"; shift 2 ;;
    --ports) PORTS="${2:?}"; shift 2 ;;
    --bucket) BUCKET_SEC="${2:?}"; shift 2 ;;
    -*) echo "Неизвестная опция: $1" >&2; usage >&2; exit 2 ;;
    *)
      if [[ -z "$PCAP" ]]; then
        PCAP="$1"
      else
        echo "Лишний аргумент: $1" >&2; exit 2
      fi
      shift
      ;;
  esac
done

if [[ -z "$PCAP" ]]; then
  echo "Укажите файл pcap." >&2
  usage >&2
  exit 2
fi
if [[ ! -f "$PCAP" ]]; then
  echo "Файл не найден: $PCAP" >&2
  exit 1
fi

if ! command -v tshark >/dev/null 2>&1; then
  echo "Нужен tshark (Wireshark CLI)." >&2
  echo "  sudo apt install tshark" >&2
  exit 1
fi
if ! command -v python3 >/dev/null 2>&1; then
  echo "Нужен python3." >&2
  exit 1
fi

# --- tshark display filter ---
TSHARK_FILTER="tcp"
if [[ -n "$HOST_FILTER" ]]; then
  TSHARK_FILTER="ip.addr == ${HOST_FILTER} && tcp"
fi
IFS=',' read -r -a PORT_ARR <<< "$PORTS"
PORT_OR=""
for p in "${PORT_ARR[@]}"; do
  p="${p// /}"
  [[ -z "$p" ]] && continue
  if [[ -n "$PORT_OR" ]]; then PORT_OR+=" or "; fi
  PORT_OR+="tcp.port == $p"
done
if [[ -n "$PORT_OR" ]]; then
  TSHARK_FILTER="($TSHARK_FILTER) && ($PORT_OR)"
fi

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

echo "=== 3wayproxy pcap analysis ==="
echo "file:   $PCAP"
echo "filter: $TSHARK_FILTER"
echo "ports:  ${PORT_ARR[*]}"
echo "bucket: ${BUCKET_SEC}s"
echo

# --- сводка по файлу ---
FIRST_TS="$(tshark -r "$PCAP" -T fields -e frame.time_epoch 2>/dev/null | head -1 || true)"
LAST_TS="$(tshark -r "$PCAP" -T fields -e frame.time_epoch 2>/dev/null | tail -1 || true)"
PKT_TOTAL="$(tshark -r "$PCAP" -q -z io,stat,0 2>/dev/null | awk '/Interval/{getline; print $5; exit}' || echo "?")"
DUR="?"
if [[ -n "$FIRST_TS" && -n "$LAST_TS" ]]; then
  DUR="$(python3 -c "print(f'{float('$LAST_TS') - float('$FIRST_TS'):.1f}')" 2>/dev/null || echo "?")"
fi
echo "--- Общее ---"
echo "длительность захвата: ${DUR}s"
echo "пакетов (всего):      ${PKT_TOTAL}"
echo

# --- TLS ClientHello / JA3 ---
echo "--- TLS ClientHello (JA3) ---"
JA3_LINES="$(tshark -r "$PCAP" -Y "ssl.handshake.type == 1" \
  -T fields -e frame.time_relative -e ip.src -e ip.dst -e tls.handshake.extensions_server_name \
  -e tls.handshake.ja3_full 2>/dev/null | sed '/^\s*$/d' || true)"
if [[ -z "$JA3_LINES" ]]; then
  # старые версии tshark
  JA3_LINES="$(tshark -r "$PCAP" -Y "ssl.handshake.type == 1" \
    -T fields -e frame.time_relative -e ip.src -e ip.dst -e tls.handshake.extensions_server_name \
    -e tls.handshake.ja3 2>/dev/null | sed '/^\s*$/d' || true)"
fi
if [[ -z "$JA3_LINES" ]]; then
  echo "(ClientHello не найден — нет TLS в захвате или только серверная сторона)"
else
  echo "time(s)  src              dst              SNI                 JA3"
  echo "$JA3_LINES" | while IFS=$'\t' read -r t src dst sni ja3; do
    printf "%6s  %-15s  %-15s  %-18s  %s\n" "$t" "$src" "$dst" "${sni:--}" "${ja3:--}"
  done
  JA3_UNIQUE="$(echo "$JA3_LINES" | awk -F'\t' '{print $NF}' | sort -u | wc -l)"
  echo
  echo "уникальных JA3: $JA3_UNIQUE  (Chrome обычно 1 профиль; Go/net/http — другой)"
fi
echo

# --- HTTP upgrade на /ws/play ---
echo "--- WebSocket upgrade (HTTP) ---"
WS_UP="$(tshark -r "$PCAP" -Y 'http.request.method == "GET" && http.request.uri contains "ws"' \
  -T fields -e frame.time_relative -e ip.src -e tcp.dstport -e http.request.uri 2>/dev/null \
  | sed '/^\s*$/d' || true)"
if [[ -z "$WS_UP" ]]; then
  echo "(GET /ws/... не найден — возможно только TLS payload или wss без расшифровки)"
else
  echo "time(s)  src              dst_port  URI"
  echo "$WS_UP" | while IFS=$'\t' read -r t src port uri; do
    printf "%6s  %-15s  %-8s  %s\n" "$t" "$src" "$port" "$uri"
  done
fi
echo

# --- TCP сегменты по портам relay ---
TCP_CSV="$WORKDIR/tcp_segments.csv"
tshark -r "$PCAP" -Y "$TSHARK_FILTER" \
  -T fields \
  -e frame.time_epoch \
  -e ip.src \
  -e ip.dst \
  -e tcp.srcport \
  -e tcp.dstport \
  -e tcp.stream \
  -e tcp.len \
  -E header=y -E separator=, -E quote=d \
  > "$TCP_CSV" 2>/dev/null || true

if [[ ! -s "$TCP_CSV" ]]; then
  echo "Нет TCP трафика по фильтру. Проверьте --host / --ports." >&2
  exit 1
fi

python3 - "$TCP_CSV" "${PORT_ARR[*]}" "$BUCKET_SEC" <<'PY'
import csv
import math
import sys
from collections import defaultdict

path = sys.argv[1]
ports = {int(p.strip()) for p in sys.argv[2].split(",") if p.strip()}
bucket = float(sys.argv[3])
if bucket <= 0:
    bucket = 1.0

rows = []
with open(path, newline="") as f:
    reader = csv.DictReader(f)
    for r in reader:
        try:
            ts = float(r["frame.time_epoch"])
            sp = int(r["tcp.srcport"])
            dp = int(r["tcp.dstport"])
            length = int(r["tcp.len"] or 0)
            stream = int(r["tcp.stream"])
        except (KeyError, ValueError, TypeError):
            continue
        if dp in ports:
            leg = dp
        elif sp in ports:
            leg = sp
        else:
            continue
        rows.append((ts, leg, length, stream))

if not rows:
    print("Нет сегментов с payload на указанных портах.")
    sys.exit(0)

t0 = min(r[0] for r in rows)
t1 = max(r[0] for r in rows)
duration = max(t1 - t0, 1e-9)

def pearson(a, b):
    n = min(len(a), len(b))
    if n < 3:
        return None
    a, b = a[:n], b[:n]
    ma = sum(a) / n
    mb = sum(b) / n
    num = sum((x - ma) * (y - mb) for x, y in zip(a, b))
    da = math.sqrt(sum((x - ma) ** 2 for x in a))
    db = math.sqrt(sum((y - mb) ** 2 for y in b))
    if da == 0 or db == 0:
        return None
    return num / (da * db)

def pct(vals, p):
    if not vals:
        return 0
    s = sorted(vals)
    i = min(len(s) - 1, max(0, int(round((p / 100.0) * (len(s) - 1)))))
    return s[i]

# bytes per leg
bytes_leg = defaultdict(int)
pkts_leg = defaultdict(int)
seg_sizes = defaultdict(list)
stream_info = defaultdict(lambda: {"leg": None, "tmin": None, "tmax": None, "bytes": 0, "pkts": 0})

for ts, leg, length, stream in rows:
    bytes_leg[leg] += length
    pkts_leg[leg] += 1
    if length > 0:
        seg_sizes[leg].append(length)
    si = stream_info[stream]
    si["leg"] = leg
    si["tmin"] = ts if si["tmin"] is None else min(si["tmin"], ts)
    si["tmax"] = ts if si["tmax"] is None else max(si["tmax"], ts)
    si["bytes"] += length
    si["pkts"] += 1

print("--- TCP payload по ногам relay ---")
total_bytes = sum(bytes_leg.values()) or 1
for leg in sorted(ports):
    b = bytes_leg.get(leg, 0)
    p = pkts_leg.get(leg, 0)
    avg_bps = b / duration
    share = 100.0 * b / total_bytes
    sizes = seg_sizes.get(leg, [])
    print(f"порт {leg}: {b} bytes, {p} сегментов, ~{avg_bps:.0f} B/s ({share:.1f}% трафика)")
    if sizes:
        print(
            f"         размер сегмента (payload): "
            f"min={min(sizes)} p50={pct(sizes,50)} p95={pct(sizes,95)} max={max(sizes)}"
        )
print()

# sessions per leg
print("--- Длительность TCP-потоков (по tcp.stream) ---")
by_leg_streams = defaultdict(list)
for sid, si in stream_info.items():
    if si["tmin"] is None:
        continue
    dur = si["tmax"] - si["tmin"]
    by_leg_streams[si["leg"]].append((dur, si["bytes"], sid))

for leg in sorted(ports):
    streams = by_leg_streams.get(leg, [])
    if not streams:
        print(f"порт {leg}: потоков не найдено")
        continue
    durs = [s[0] for s in streams]
    print(
        f"порт {leg}: {len(streams)} поток(ов), "
        f"длительность min={min(durs):.2f}s med={pct(durs,50):.2f}s max={max(durs):.2f}s"
    )
long_all = [d for streams in by_leg_streams.values() for d, _, _ in streams]
if long_all:
    over_60 = sum(1 for d in long_all if d >= 60)
    print(f"всего потоков: {len(long_all)}, из них >=60s: {over_60}  (долгие WS = сигнал туннеля)")
print()

# time buckets for correlation
bucket_idx = lambda ts: int((ts - t0) / bucket)
buckets = defaultdict(lambda: defaultdict(int))
for ts, leg, length, _ in rows:
    buckets[bucket_idx(ts)][leg] += length

max_idx = max(buckets) if buckets else 0
series = {leg: [buckets[i].get(leg, 0) for i in range(max_idx + 1)] for leg in sorted(ports)}

print(f"--- Корреляция bytes/{bucket:g}s между ногами (Пирсон) ---")
legs = sorted(ports)
pairs = []
for i, a in enumerate(legs):
    for b in legs[i + 1 :]:
        r = pearson(series[a], series[b])
        pairs.append((a, b, r))
        if r is None:
            print(f"  {a} ↔ {b}: недостаточно данных")
        else:
            print(f"  {a} ↔ {b}: r = {r:.3f}")

if pairs:
    vals = [abs(p[2]) for p in pairs if p[2] is not None]
    if vals:
        avg_abs = sum(vals) / len(vals)
        print()
        if avg_abs >= 0.85:
            hint = "высокая — три ноги растут синхронно (типичный признак одного туннеля)"
        elif avg_abs >= 0.55:
            hint = "средняя — частичная синхронизация; padding/jitter/churn должны снижать"
        else:
            hint = "низкая — потоки слабо связаны по времени (лучше для маскировки)"
        print(f"средняя |r|: {avg_abs:.3f} — {hint}")
print()

# simultaneous connect windows (streams starting within 2s)
print("--- Синхронные старты TCP (окно 2s) ---")
starts = [(si["tmin"], si["leg"], sid) for sid, si in stream_info.items() if si["tmin"] is not None]
starts.sort()
clusters = []
for t, leg, sid in starts:
    if not clusters or t - clusters[-1][0] > 2.0:
        clusters.append([t, {leg}])
    else:
        clusters[-1][1].add(leg)
sync3 = sum(1 for _, legs in clusters if len(legs) >= 3)
sync2 = sum(1 for _, legs in clusters if len(legs) == 2)
print(f"кластеров старта: {len(clusters)}, с 2 ногами: {sync2}, с 3 ногами: {sync3}")
if sync3 > 0:
    print("  ⚠ три ноги стартуют почти одновременно — заметно при поведенческом анализе")
print()

# scorecard
print("--- Чеклист (эвристики, не гарантия) ---")
checks = []

# balance
shares = [bytes_leg.get(leg, 0) / total_bytes for leg in sorted(ports)]
imb = max(shares) - min(shares) if shares else 0
checks.append(("баланс трафика между ногами (<15% разброс)", imb < 0.15, f"разброс долей {imb*100:.1f}%"))

if vals:
    checks.append(("корреляция |r| < 0.55", avg_abs < 0.55, f"avg |r|={avg_abs:.3f}"))

if long_all:
    checks.append(("мало потоков >=60s", over_60 <= len(long_all) // 3, f"{over_60}/{len(long_all)} длинных"))

checks.append(("синхронный старт 3 ног редок", sync3 == 0, f"sync3={sync3}"))

for name, ok, detail in checks:
    mark = "OK" if ok else "!!"
    print(f"  [{mark}] {name} ({detail})")

print()
print("Подсказка: сравните с эталоном — pcap обычной браузерной игры на тех же портах.")
print("Документация: docs/PLAN.md (фазы padding/jitter/decoy).")
PY

echo
echo "=== Готово ==="
echo "Повторный захват после изменений (rotate/churn/padding) — сравните корреляцию и длительность потоков."
