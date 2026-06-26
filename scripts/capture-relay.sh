#!/usr/bin/env bash
# Захват pcap на relay-сервере (порты 81–83).
set -euo pipefail
OUT="${1:-capture-relay-$(date +%Y%m%d-%H%M%S).pcap}"
echo "Пишем $OUT (Ctrl+C для остановки)"
echo "Фильтр: tcp port 81 or 82 or 83"
sudo tcpdump -ni any 'tcp port 81 or tcp port 82 or tcp port 83' -w "$OUT"
