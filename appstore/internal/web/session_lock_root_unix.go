//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package web

import (
	"os/user"
	"path/filepath"
	"strings"
)

func platformSessionLockRoot() string {
	// Use the account database instead of HOME or TMPDIR so processes that
	// reach the same per-user keychain cannot select different lock anchors.
	current, err := user.Current()
	if err != nil {
		return ""
	}
	home := strings.TrimSpace(current.HomeDir)
	if home == "" || !filepath.IsAbs(home) {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(home)
	if err != nil {
		return ""
	}
	return resolved
}

func platformSessionLockDirName() string { return ".asc-web-session-locks" }
