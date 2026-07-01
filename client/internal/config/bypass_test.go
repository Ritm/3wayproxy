package config

import "testing"

func TestBootstrapBypassIncludesRelayNotDestinations(t *testing.T) {
	cfg := &Config{
		Relays: []RelayConfig{{
			WS: "wss://relay-ritm.amvera.io/ws/play",
		}},
		TUN: TUN{
			Routes:       []string{"8.8.8.8", "188.40.167.82"},
			BypassRoutes: []string{"82.40.62.35"},
		},
	}
	ips, err := cfg.BootstrapBypass()
	if err != nil {
		t.Fatal(err)
	}
	has := func(ip string) bool {
		for _, x := range ips {
			if x == ip {
				return true
			}
		}
		return false
	}
	if !has("82.40.62.35") {
		t.Fatalf("missing manual bypass: %v", ips)
	}
	if !has("81.26.184.189") {
		t.Fatalf("missing resolved relay IP: %v", ips)
	}
	if has("188.40.167.82") {
		t.Fatalf("destination IP must not be bypassed: %v", ips)
	}
	if has("8.8.8.8") {
		t.Fatalf("tun.routes IP must not auto-bypass: %v", ips)
	}
}
