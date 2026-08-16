//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package xcode

import (
	"os"
	"syscall"
)

func currentExactIPAOwner() (uint64, bool) {
	return uint64(os.Geteuid()), true
}

func exactIPAStatIdentity(info os.FileInfo) (uid, nlink uint64, ok bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return 0, 0, false
	}
	return uint64(stat.Uid), uint64(stat.Nlink), true
}
