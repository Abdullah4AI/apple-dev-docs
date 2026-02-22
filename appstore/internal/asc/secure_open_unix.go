//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package asc

import (
	"os"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

func openExistingNoFollow(path string) (*os.File, error) {
	return secureopen.OpenExistingNoFollow(path)
}
