#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=lib/relay-venv.sh
source "$SCRIPT_DIR/lib/relay-venv.sh"
relay_ensure_venv

# Без --reload: перезагрузка рвёт WebSocket и ломает туннель.
exec "$RELAY_PY" -m uvicorn app.main:app --host 0.0.0.0 --port 8000
