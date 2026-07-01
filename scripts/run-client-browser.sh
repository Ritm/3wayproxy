#!/usr/bin/env bash
# Client with browser carrier: preserve user HOME for Playwright under sudo.
set -euo pipefail
cd "$(dirname "$0")/.."

CFG="${1:-config/client.dev.3relay.yaml}"

if ip link show tun3way &>/dev/null; then
  echo "WARN: tun3way уже существует — остановите старый client (Ctrl+C) или:"
  echo "  sudo ip link del tun3way"
  echo ""
fi

if [[ -n "${SUDO_USER:-}" && "$(id -u)" -ne 0 ]]; then
  exec sudo -E HOME="${HOME}" XDG_CACHE_HOME="${HOME}/.cache" \
    "$0" "$CFG"
fi

exec ./bin/3wayproxy-client --config "$CFG"
