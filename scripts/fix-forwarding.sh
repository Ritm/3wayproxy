#!/usr/bin/env bash
# Одноразовая настройка FORWARD/NAT для dev (нужен sudo).
set -euo pipefail
EGRESS="${1:-enp4s0}"
TUN="${2:-tun3agg}"

echo "egress=$EGRESS tun=$TUN"

sudo sysctl -w net.ipv4.ip_forward=1
sudo sysctl -w net.ipv4.conf.all.rp_filter=0
sudo sysctl -w net.ipv4.conf.default.rp_filter=0
sudo sysctl -w "net.ipv4.conf.${EGRESS}.rp_filter=0"
sudo sysctl -w "net.ipv4.conf.${TUN}.rp_filter=0" 2>/dev/null || true

sudo iptables -P FORWARD ACCEPT

sudo iptables -I FORWARD 1 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
sudo iptables -I FORWARD 1 -i "$TUN" -o "$EGRESS" -j ACCEPT
sudo iptables -I FORWARD 1 -i "$EGRESS" -o "$TUN" -j ACCEPT

sudo iptables -t nat -I POSTROUTING 1 -s 10.0.0.2/32 -o "$EGRESS" -j MASQUERADE

if sudo iptables -L DOCKER-USER -n &>/dev/null; then
  sudo iptables -I DOCKER-USER 1 -i "$TUN" -o "$EGRESS" -j ACCEPT
  sudo iptables -I DOCKER-USER 1 -i "$EGRESS" -o "$TUN" -j ACCEPT
fi

if command -v ufw >/dev/null && sudo ufw status 2>/dev/null | grep -qi "active"; then
  sudo ufw route allow in on "$TUN" out on "$EGRESS" 2>/dev/null || true
  sudo ufw route allow in on "$EGRESS" out on "$TUN" 2>/dev/null || true
else
  echo "ufw: пропуск (не активен или недоступен)"
fi

echo "OK. Проверка: sudo iptables -t nat -L POSTROUTING -n -v | head -5"
