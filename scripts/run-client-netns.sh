#!/usr/bin/env bash
# Client в отдельном network namespace — обязательно, если aggregator на том же хосте.
# Иначе ответы на 10.0.0.2 уходят в tun3way (client), а не в tun3agg (aggregator).
set -euo pipefail
cd "$(dirname "$0")/.."

NETNS="${NETNS:-3wayclient}"

if ! sudo ip netns list | grep -q "^${NETNS}"; then
  sudo ip netns add "$NETNS"
fi
sudo ip netns exec "$NETNS" ip link set lo up

# Убрать tun3way из root ns, если остался от прошлого запуска без netns
if ip link show tun3way &>/dev/null; then
  echo "WARN: tun3way в root namespace — удаляем (конфликт с aggregator)"
  sudo ip link del tun3way 2>/dev/null || true
fi

echo "Запуск client в netns=$NETNS"
echo "Проверка ping: sudo ip netns exec $NETNS ping -c 3 8.8.8.8"
exec sudo ip netns exec "$NETNS" ./bin/3wayproxy-client --config config/client.dev.yaml
