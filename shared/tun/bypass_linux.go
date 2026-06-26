//go:build linux

package tun

import (
	"fmt"
	"os/exec"
	"strings"
)

// AddBypassRoutes installs host routes via the default gateway (not via TUN).
// Use for relay server IPs so WebSocket stays direct while other traffic uses TUN.
func AddBypassRoutes(hosts []string) error {
	gw, dev, err := defaultRoute()
	if err != nil {
		return err
	}
	for _, h := range hosts {
		h = strings.TrimSuffix(h, "/32")
		if h == "" {
			continue
		}
		dst := h + "/32"
		out, err := exec.Command("ip", "route", "replace", dst, "via", gw, "dev", dev).CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "File exists") {
				continue
			}
			return fmt.Errorf("ip route bypass %s: %w: %s", dst, err, out)
		}
	}
	return nil
}

func defaultRoute() (gw, dev string, err error) {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return "", "", fmt.Errorf("ip route default: %w", err)
	}
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "via" && i+1 < len(fields) {
			gw = fields[i+1]
		}
		if f == "dev" && i+1 < len(fields) {
			dev = fields[i+1]
		}
	}
	if gw == "" || dev == "" {
		return "", "", fmt.Errorf("cannot parse default route: %q", string(out))
	}
	return gw, dev, nil
}
