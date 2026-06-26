# shellcheck shell=bash
# Подключение: source "$(dirname "$0")/lib/relay-venv.sh" && relay_ensure_venv
relay_ensure_venv() {
  local root
  root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../relay" && pwd)"
  cd "$root"

  if [[ -d .venv ]] && ! .venv/bin/python3 -c "import sys" 2>/dev/null; then
    echo "relay: удаляем битый .venv (часто после scp с другого хоста)..."
    rm -rf .venv
  fi

  if [[ ! -x .venv/bin/python3 ]]; then
    if ! python3 -m venv --help &>/dev/null; then
      echo "relay: нужен python3-venv (sudo apt install python3-venv python3-pip)"
      exit 1
    fi
    python3 -m venv .venv
    .venv/bin/python3 -m pip install --upgrade pip
    .venv/bin/python3 -m pip install -r requirements.txt
  fi

  if ! .venv/bin/python3 -m uvicorn --help &>/dev/null; then
    .venv/bin/python3 -m pip install -r requirements.txt
  fi

  RELAY_PY="$root/.venv/bin/python3"
}
