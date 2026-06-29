#!/usr/bin/env bash
# Запуск одного relay (Node.js). Порт: PORT или 8000.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/relay-node"
if [[ ! -d node_modules ]]; then
  npm install --omit=dev
fi
export RELAY_SHARD_ID="${RELAY_SHARD_ID:-0}"
export PORT="${PORT:-8000}"
exec node src/server.js
