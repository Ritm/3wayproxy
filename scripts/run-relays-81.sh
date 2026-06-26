#!/usr/bin/env bash
# Три relay на портах 81–83 (публичный деплой без nginx).
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=lib/relay-venv.sh
source "$SCRIPT_DIR/lib/relay-venv.sh"
relay_ensure_venv

cd "$SCRIPT_DIR/../relay"
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
  RELAY_SHARD_ID="$s" "$RELAY_PY" -m uvicorn app.main:app --host 0.0.0.0 --port "$p" &
  pid=$!
  PIDS+=("$pid")
  echo "relay shard=$s port=$p pid=$pid"
done

echo "3 relays on :81 :82 :83 (0.0.0.0)"
wait
