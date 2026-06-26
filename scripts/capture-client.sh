#!/usr/bin/env bash
# Захват pcap на клиенте (трафик к relay).
# Использование: ./scripts/capture-client.sh [host] [output.pcap]
set -euo pipefail
HOST="${1:-serv2.erofeevonline.ru}"
OUT="${2:-capture-client-$(date +%Y%m%d-%H%M%S).pcap}"
echo "Пишем $OUT — хост $HOST (Ctrl+C для остановки)"
sudo tcpdump -ni any host "$HOST" -w "$OUT"
