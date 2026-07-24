#!/usr/bin/env bash
# Синхронизирует relay → deploy/amvera/ (источник для GitHub→Amvera)
# и опционально собирает плоскую папку для ручной загрузки.
#
#   bash deploy/amvera/pack.sh              # sync + deploy/amvera-upload/
#   bash deploy/amvera/pack.sh /tmp/out     # sync + указанная папка
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
STAGING="$ROOT/deploy/amvera"
OUT="${1:-$ROOT/deploy/amvera-upload}"

mkdir -p "$STAGING/app" "$STAGING/game"
cp -a "$ROOT/relay/app/." "$STAGING/app/"
cp -a "$ROOT/game/." "$STAGING/game/"
find "$STAGING" -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true

rm -rf "$OUT"
mkdir -p "$OUT"
cp -a "$STAGING/app" "$STAGING/game" "$OUT/"
cp "$STAGING/run.py" "$STAGING/requirements.txt" "$STAGING/amvera.yml" "$OUT/"
find "$OUT" -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true

echo "Синхронизировано: $STAGING/{app,game}"
echo "Плоский пакет:    $OUT"
echo "Для GitHub→Amvera: закоммитьте deploy/amvera/ и корневой amvera.yml, затем git push origin main"
find "$OUT" -maxdepth 2 -type f -o -type d | sort | head -30
