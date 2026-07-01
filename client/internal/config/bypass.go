package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// BootstrapBypass returns IPs that must reach the Internet directly (not via TUN):
// explicit bypass_routes plus resolved relay hostnames for WebSocket carrier.
// Destination IPs from tun.routes are NOT bypassed — those should use the tunnel.
func (c *Config) BootstrapBypass() ([]string, error) {
	seen := make(map[string]struct{})
	var out []string
	add := func(ip string) {
		ip = strings.TrimSuffix(ip, "/32")
		if ip == "" || net.ParseIP(ip) == nil {
			return
		}
		if _, ok := seen[ip]; ok {
			return
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}

	for _, h := range c.TUN.BypassRoutes {
		add(h)
	}
	for _, ep := range c.endpoints() {
		u, err := url.Parse(ep.URL)
		if err != nil {
			return nil, fmt.Errorf("relay url %q: %w", ep.URL, err)
		}
		host := u.Hostname()
		if host == "" {
			continue
		}
		if ip := net.ParseIP(host); ip != nil {
			add(host)
			continue
		}
		addrs, err := net.LookupHost(host)
		if err != nil {
			return nil, fmt.Errorf("resolve relay host %q: %w", host, err)
		}
		for _, a := range addrs {
			add(a)
		}
	}
	return out, nil
}
