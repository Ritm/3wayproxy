//go:build !linux

package tun

import "fmt"

func RouteDev(dst string) (string, error) {
	return "", fmt.Errorf("RouteDev: not implemented")
}
