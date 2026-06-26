#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
sudo ./bin/3wayproxy-agg --config config/aggregator.dev.yaml
