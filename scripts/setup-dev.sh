#!/usr/bin/env bash
# Build binaries and install Python deps for Ubuntu 24.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1; then
  echo "Installing golang-go..."
  sudo apt-get update -qq
  sudo apt-get install -y golang-go
fi

if ! python3 -m venv --help >/dev/null 2>&1; then
  echo "Installing python3-venv..."
  sudo apt-get update -qq
  sudo apt-get install -y python3-venv python3-pip
fi

export GOFLAGS="${GOFLAGS:--buildvcs=false}"

echo "Go modules..."
(cd shared && go mod tidy)
(cd client && go mod tidy)
(cd aggregator && go mod tidy)

make build
make test

# Не копируйте relay/.venv через scp — shebang указывает на Python другой машины.
if [[ -d relay/.venv ]]; then
  if ! relay/.venv/bin/python3 -c "import sys" 2>/dev/null; then
    echo "Удаляем битый relay/.venv (часто после scp с другого хоста)..."
    rm -rf relay/.venv
  fi
fi

python3 -m venv relay/.venv
relay/.venv/bin/python3 -m pip install --upgrade pip
relay/.venv/bin/python3 -m pip install -r relay/requirements.txt

chmod +x scripts/*.sh
echo "Done. Start: scripts/run-relay.sh, then scripts/run-aggregator.sh, then scripts/run-client.sh"
