//go:build linux

package nat

import (
	"fmt"
	"os/exec"
	"strings"
)

// Setup enables forwarding, MASQUERADE and FORWARD rules for tun ↔ egress.
// tunDev: e.g. tun3agg. peerNet: client address, e.g. 10.0.0.2
func Setup(tunDev, egress, peerNet string, permissive bool) (string, error) {
	if egress == "" {
		var err error
		egress, err = defaultRouteInterface()
		if err != nil {
			return "", err
		}
	}
	if tunDev == "" {
		tunDev = "tun+"
	}
	if peerNet == "" {
		peerNet = "10.0.0.2"
	}
	peerNet = strings.TrimSuffix(peerNet, "/32")

	// rp_filter=0: иначе ответы с enp4s0 часто дропаются при dst=10.0.0.2
	sysctls := []string{
		"net.ipv4.ip_forward=1",
		"net.ipv4.conf.all.forwarding=1",
		"net.ipv4.conf.all.rp_filter=0",
		"net.ipv4.conf.default.rp_filter=0",
		fmt.Sprintf("net.ipv4.conf.%s.rp_filter=0", egress),
	}
	if tunDev != "tun+" {
		sysctls = append(sysctls, fmt.Sprintf("net.ipv4.conf.%s.rp_filter=0", tunDev))
		sysctls = append(sysctls, fmt.Sprintf("net.ipv4.conf.%s.forwarding=1", tunDev))
	}
	for _, key := range sysctls {
		if out, err := exec.Command("sysctl", "-w", key).CombinedOutput(); err != nil {
			// tunX может ещё не существовать в sysctl до up — не фатально
			if strings.Contains(key, "tun") {
				continue
			}
			return "", fmt.Errorf("sysctl %s: %w: %s", key, err, out)
		}
	}

	if permissive {
		if out, err := exec.Command("iptables", "-P", "FORWARD", "ACCEPT").CombinedOutput(); err != nil {
			return "", fmt.Errorf("iptables -P FORWARD ACCEPT: %w: %s", err, out)
		}
	}

	// FORWARD: в начало цепочки
	fwd := [][]string{
		{"-I", "FORWARD", "1", "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
		{"-I", "FORWARD", "1", "-i", tunDev, "-o", egress, "-j", "ACCEPT"},
		{"-I", "FORWARD", "1", "-i", egress, "-o", tunDev, "-j", "ACCEPT"},
	}
	for _, rule := range fwd {
		if err := ipt(rule); err != nil {
			return "", err
		}
	}

	// MASQUERADE: -I в POSTROUTING (критично — без NAT ответы на 10.0.0.2 не придут)
	natRules := [][]string{
		{"-t", "nat", "-I", "POSTROUTING", "1", "-s", peerNet + "/32", "-o", egress, "-j", "MASQUERADE"},
		{"-t", "nat", "-I", "POSTROUTING", "1", "-s", "10.0.0.0/30", "-o", egress, "-j", "MASQUERADE"},
	}
	for _, rule := range natRules {
		if err := ipt(rule); err != nil {
			return "", err
		}
	}

	// Docker часто дропает FORWARD до наших правил
	if chainExists("DOCKER-USER") {
		for _, rule := range [][]string{
			{"-I", "DOCKER-USER", "1", "-i", tunDev, "-o", egress, "-j", "ACCEPT"},
			{"-I", "DOCKER-USER", "1", "-i", egress, "-o", tunDev, "-j", "ACCEPT"},
		} {
			_ = ipt(rule)
		}
	}

	// UFW (если установлен) — не фатально
	_, _ = exec.Command("ufw", "route", "allow", "in", "on", tunDev, "out", "on", egress).CombinedOutput()
	_, _ = exec.Command("ufw", "route", "allow", "in", "on", egress, "out", "on", tunDev).CombinedOutput()

	return egress, nil
}

func ipt(rule []string) error {
	args := append([]string{"iptables"}, rule...)
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		msg := string(out)
		if strings.Contains(msg, "already exists") {
			return nil
		}
		return fmt.Errorf("iptables %v: %w: %s", rule, err, out)
	}
	return nil
}

func chainExists(name string) bool {
	out, err := exec.Command("iptables", "-L", name, "-n").CombinedOutput()
	return err == nil && strings.Contains(string(out), name)
}

func defaultRouteInterface() (string, error) {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return "", fmt.Errorf("ip route: %w", err)
	}
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	return "", fmt.Errorf("no default route interface found")
}
