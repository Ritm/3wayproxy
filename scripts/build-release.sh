#!/usr/bin/env bash
# Сборка статических бинарников для Linux (client + aggregator).
# Использование:
#   ./scripts/build-release.sh              # amd64 (по умолчанию)
#   ./scripts/build-release.sh arm64        # для Raspberry Pi / ARM VPS
#   ./scripts/build-release.sh amd64 arm64  # оба
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1; then
  if [[ -x "$ROOT/.tools/go/bin/go" ]]; then
    export PATH="$ROOT/.tools/go/bin:$PATH"
    export GOTOOLCHAIN=local
  else
    echo "Go не найден. Установите: sudo apt install golang-go"
    exit 1
  fi
fi

archs=("$@")
if [[ ${#archs[@]} -eq 0 ]]; then
  archs=(amd64)
fi

mkdir -p dist

for arch in "${archs[@]}"; do
  case "$arch" in
    amd64|arm64|386|arm) ;;
    *)
      echo "Неизвестная архитектура: $arch (допустимо: amd64 arm64 386 arm)"
      exit 1
      ;;
  esac
  echo "==> GOOS=linux GOARCH=$arch"
  export CGO_ENABLED=0
  export GOOS=linux
  export GOARCH="$arch"
  export GOFLAGS="-buildvcs=false"
  suffix=""
  [[ "$arch" != "amd64" ]] && suffix="-linux-${arch}"
  (cd client && go build -trimpath -ldflags="-s -w" -o "../dist/3wayproxy-client${suffix}" ./cmd/3wayproxy-client)
  (cd aggregator && go build -trimpath -ldflags="-s -w" -o "../dist/3wayproxy-agg${suffix}" ./cmd/3wayproxy-agg)
  file "dist/3wayproxy-client${suffix}" "dist/3wayproxy-agg${suffix}"
done

echo
echo "Готово: dist/"
ls -la dist/
