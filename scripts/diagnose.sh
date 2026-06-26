#!/usr/bin/env bash
# Диагностика туннеля (нужен sudo для iptables/sysctl).
set -euo pipefail
echo "=== routes ==="
ip route get 8.8.8.8 2>/dev/null || true
ip route show table main | grep -E 'tun3|8\.8\.8|1\.1\.1' || true
echo
echo "=== tun interfaces ==="
ip -br addr show tun3way tun3agg 2>/dev/null || true
echo
echo "=== forwarding ==="
sysctl net.ipv4.ip_forward net.ipv4.conf.all.rp_filter 2>/dev/null || sudo sysctl net.ipv4.ip_forward net.ipv4.conf.all.rp_filter
echo
echo "=== iptables FORWARD (first 15) ==="
sudo iptables -L FORWARD -n -v --line-numbers 2>/dev/null | head -15 || echo "(need sudo)"
echo
echo "=== iptables NAT POSTROUTING (должны расти счётчики pkts при ping) ==="
sudo iptables -t nat -L POSTROUTING -n -v --line-numbers 2>/dev/null | head -10 || true
echo
echo "=== tcpdump hint ==="
echo "sudo tcpdump -ni enp4s0 icmp and host 8.8.8.8"
echo "  → должны быть echo request (src 192.168.x.x) И echo reply"
echo "  → если request с src 10.0.0.2 — MASQUERADE не работает"
echo
echo "=== relay ==="
curl -sf http://127.0.0.1:8000/health && echo " relay ok" || echo "relay NOT running"
echo
echo "=== ufw ==="
if command -v ufw >/dev/null; then sudo ufw status 2>/dev/null | head -5 || true; fi
echo
echo "Если ping не идёт, но client пишет 'tun read' — проверьте FORWARD/NAT выше."
echo "При активном UFW: sudo ufw route allow in on tun3agg out on enp4s0"
