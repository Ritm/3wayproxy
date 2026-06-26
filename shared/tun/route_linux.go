//go:build linux

package tun

import (
	"fmt"
	"os/exec"
	"strings"
)

// EnsurePeerRoute makes sure packets to peerIP are delivered via dev.
func EnsurePeerRoute(dev, peerIP string) error {
	peerIP = strings.TrimSuffix(peerIP, "/32")
	if peerIP == "" {
		return nil
	}
	dst := peerIP + "/32"
	out, err := exec.Command("ip", "route", "replace", dst, "dev", dev).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "File exists") {
			return nil
		}
		return fmt.Errorf("ip route replace %s dev %s: %w: %s", dst, dev, err, out)
	}
	return nil
}

// PeerRouteDev returns the interface the kernel would use to reach peerIP.
func PeerRouteDev(peerIP string) (string, error) {
	peerIP = strings.TrimSuffix(peerIP, "/32")
	out, err := exec.Command("ip", "route", "get", peerIP).Output()
	if err != nil {
		return "", fmt.Errorf("ip route get %s: %w", peerIP, err)
	}
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	return "", fmt.Errorf("cannot parse route get for %s: %q", peerIP, string(out))
}
