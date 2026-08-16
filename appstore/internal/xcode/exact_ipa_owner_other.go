//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly

package xcode

import "os"

func currentExactIPAOwner() (uint64, bool) {
	return 0, false
}

func exactIPAStatIdentity(os.FileInfo) (uid, nlink uint64, ok bool) {
	return 0, 0, false
}
