//go:build linux

package tun

import (
	"fmt"
	"os/exec"
	"strings"
)

// RouteDev returns the interface the kernel would use to reach dst (host or CIDR).
func RouteDev(dst string) (string, error) {
	dst = strings.TrimSpace(dst)
	if dst == "" {
		return "", fmt.Errorf("empty destination")
	}
	if !strings.Contains(dst, "/") {
		dst = dst + "/32"
	}
	out, err := exec.Command("ip", "route", "get", strings.Fields(dst)[0]).Output()
	if err != nil {
		return "", fmt.Errorf("ip route get %s: %w", dst, err)
	}
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	return "", fmt.Errorf("cannot parse route get for %s: %q", dst, string(out))
}
