//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package distribute

import (
	"os"
	"syscall"
)

func archiveHasMultipleHardLinks(info os.FileInfo) (multiple, supported bool) {
	if info == nil {
		return false, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return false, false
	}
	return uint64(stat.Nlink) > 1, true
}
