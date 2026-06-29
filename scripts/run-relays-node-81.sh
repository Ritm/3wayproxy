#!/usr/bin/env bash
# Три Node relay на портах 81–83 (деплой без nginx).
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT/relay-node"
npm install --omit=dev

PIDS=()
cleanup() {
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

for entry in 81:0 82:1 83:2; do
  p="${entry%%:*}"
  s="${entry##*:}"
  RELAY_SHARD_ID="$s" PORT="$p" node src/server.js &
  pid=$!
  PIDS+=("$pid")
  echo "relay-node shard=$s port=$p pid=$pid"
done

echo "3 relay-node: :81 :82 :83"
wait
