#!/usr/bin/env bash
# Локальная сборка папки для загрузки на Amvera (запускать на ПК, не на хостинге).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="${1:-$ROOT/deploy/amvera-upload}"

rm -rf "$OUT"
mkdir -p "$OUT/app" "$OUT/game"

cp -a "$ROOT/relay/app/." "$OUT/app/"
cp -a "$ROOT/game/." "$OUT/game/"
cp "$ROOT/deploy/amvera/run.py" \
   "$ROOT/deploy/amvera/requirements.txt" \
   "$ROOT/deploy/amvera/amvera.yml" \
   "$OUT/"

echo "Готово: $OUT"
echo "Загрузите ВСЁ содержимое этой папки в корень Code на Amvera:"
find "$OUT" -maxdepth 2 -type f -o -type d | sort | head -30
