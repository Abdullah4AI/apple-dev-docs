//go:build !darwin && !linux

package rootfs

import "os"

// sameFileOwnership has no observable meaning outside Unix because os.FileInfo
// exposes no portable owner. Identity-coupled mutations already refuse to move
// an entry on those platforms, so ownership drift cannot be preserved into a
// replacement there and the comparison stays permissive for the read-only
// identity checks that do run.
func sameFileOwnership(_, _ os.FileInfo) bool {
	return true
}
