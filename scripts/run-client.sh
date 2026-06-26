#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

if ip link show tun3agg &>/dev/null; then
  echo "WARN: aggregator (tun3agg) уже запущен на этом хосте."
  echo "      Client на той же машине → используйте ./scripts/run-client-netns.sh"
  echo "      Иначе ICMP-ответы не вернутся в aggregator."
  echo ""
fi

sudo ./bin/3wayproxy-client --config config/client.dev.yaml
