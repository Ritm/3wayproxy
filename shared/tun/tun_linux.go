//go:build linux

package tun

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/songgao/water"
)

// Config describes a point-to-point TUN interface.
type Config struct {
	Name    string
	LocalIP string
	PeerIP  string
	MTU     int
}

// Open creates and configures a TUN device.
func Open(cfg Config) (*water.Interface, error) {
	if cfg.MTU == 0 {
		cfg.MTU = 1200
	}
	wcfg := water.Config{DeviceType: water.TUN}
	if cfg.Name != "" {
		wcfg.Name = cfg.Name
	}
	iface, err := water.New(wcfg)
	if err != nil {
		return nil, fmt.Errorf("tun create: %w", err)
	}
	name := iface.Name()
	local := withPrefix32(stripCIDR(cfg.LocalIP))
	peer := withPrefix32(stripCIDR(cfg.PeerIP))
	// Аргументы раздельно: ip addr add LOCAL/32 peer REMOTE/32 dev NAME
	if out, err := exec.Command("ip", "addr", "add", local, "peer", peer, "dev", name).CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "File exists") {
			_ = iface.Close()
			return nil, fmt.Errorf("ip addr add: %w: %s", err, out)
		}
	}
	if out, err := exec.Command("ip", "link", "set", "dev", name, "mtu", fmt.Sprint(cfg.MTU)).CombinedOutput(); err != nil {
		_ = iface.Close()
		return nil, fmt.Errorf("ip link set mtu: %w: %s", err, out)
	}
	if out, err := exec.Command("ip", "link", "set", "dev", name, "up").CombinedOutput(); err != nil {
		_ = iface.Close()
		return nil, fmt.Errorf("ip link set up: %w: %s", err, out)
	}
	return iface, nil
}

// AddRoutes installs additional routes via the TUN device.
func AddRoutes(dev string, routes []string) error {
	for _, r := range routes {
		if r == "" {
			continue
		}
		dst := r
		if !strings.Contains(dst, "/") {
			dst = dst + "/32"
		}
		args := append([]string{"route", "add"}, strings.Fields(dst)...)
		args = append(args, "dev", dev)
		if out, err := exec.Command("ip", args...).CombinedOutput(); err != nil {
			if strings.Contains(string(out), "File exists") {
				continue
			}
			return fmt.Errorf("ip route add %s: %w: %s", r, err, out)
		}
	}
	return nil
}

func stripCIDR(addr string) string {
	if i := strings.Index(addr, "/"); i >= 0 {
		return addr[:i]
	}
	return addr
}

func withPrefix32(addr string) string {
	if strings.Contains(addr, "/") {
		return addr
	}
	return addr + "/32"
}
