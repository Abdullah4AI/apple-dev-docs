//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly && !windows

package rootfs

import "os"

func hasMultipleHardLinks(_ *os.File, _ os.FileInfo) (bool, error) {
	return false, nil
}
