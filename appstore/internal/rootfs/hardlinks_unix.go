//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package rootfs

import (
	"os"
	"syscall"
)

func hasMultipleHardLinks(_ *os.File, info os.FileInfo) (bool, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1, nil
}
