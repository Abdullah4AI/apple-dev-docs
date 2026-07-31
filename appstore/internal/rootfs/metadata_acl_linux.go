//go:build linux

package rootfs

import "os"

func copyAccessControlList(_, _ *os.File) error {
	// Linux POSIX ACLs are exposed as system.posix_acl_* extended attributes,
	// which copyExtendedAttributes preserves through the open descriptors.
	return nil
}
