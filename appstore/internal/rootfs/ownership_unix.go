//go:build darwin || linux

package rootfs

import (
	"os"
	"syscall"
)

// sameFileOwnership reports whether two stat results observe the same owning
// user and group. Strict identity checks need it because a concurrent chown
// changes neither the permission bits nor the modification time, yet
// copyReplacementMetadata carries the observed ownership onto the replacement.
// A stat result that cannot expose ownership cannot prove the owner is
// unchanged, so the comparison fails closed.
func sameFileOwnership(first, second os.FileInfo) bool {
	firstStat, firstOK := first.Sys().(*syscall.Stat_t)
	secondStat, secondOK := second.Sys().(*syscall.Stat_t)
	if !firstOK || !secondOK {
		return false
	}
	return firstStat.Uid == secondStat.Uid && firstStat.Gid == secondStat.Gid
}
