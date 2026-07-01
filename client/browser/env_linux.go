//go:build linux

package browser

import (
	"os"
	"os/user"
)

func prepareSudoEnv() {
	if os.Geteuid() != 0 {
		return
	}
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" || sudoUser == "root" {
		return
	}
	u, err := user.Lookup(sudoUser)
	if err != nil {
		return
	}
	_ = os.Setenv("HOME", u.HomeDir)
	_ = os.Setenv("XDG_CACHE_HOME", u.HomeDir+"/.cache")
}
