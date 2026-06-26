#!/usr/bin/env bash
# Install Playwright Chromium for 3wayproxy client (phase 3).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export GOFLAGS="${GOFLAGS:--buildvcs=false}"
export PATH="${ROOT}/.tools/go/bin:${PATH}"

echo "Installing Playwright Chromium (one-time, ~150MB)..."
cd "${ROOT}/client"
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.0 install chromium
echo "Done. Run client with carrier: browser in config."
