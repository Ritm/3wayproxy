#!/usr/bin/env bash
# Быстрая проверка relay перед добавлением в client.yaml
set -euo pipefail
BASE="${1:?Usage: $0 https://game.example.com}"
BASE="${BASE%/}"

echo "=== probe $BASE ==="

code=$(curl -sf -o /dev/null -w '%{http_code}' "$BASE/health" || echo "fail")
echo "GET /health → $code (expect 200 + ok)"
curl -sf "$BASE/health" && echo || echo "FAIL"

code=$(curl -sf -o /dev/null -w '%{http_code}' "$BASE/" || echo "fail")
echo "GET / → HTTP $code (expect 200)"

code=$(curl -sf -o /dev/null -w '%{http_code}' "$BASE/play.html" || echo "fail")
echo "GET /play.html → HTTP $code (expect 200)"

if command -v wscat >/dev/null 2>&1; then
  ws="${BASE/https:/wss:}/ws/chess"
  echo "WS $ws (3s timeout)..."
  timeout 3 wscat -c "$ws" 2>/dev/null | head -1 || echo "(wscat timeout ok if server accepts)"
else
  echo "Установите wscat для проверки WS: npm install -g wscat"
fi

echo "=== done ==="
