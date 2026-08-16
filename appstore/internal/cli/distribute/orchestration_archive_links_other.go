//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly

package distribute

import "os"

func archiveHasMultipleHardLinks(os.FileInfo) (multiple, supported bool) {
	return false, false
}
