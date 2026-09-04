//go:build linux

package certificates

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// linuxPOSIXACLAccessAttribute stores a file's extended POSIX ACL entries. The
// attribute exists only when entries beyond the owner/group/other mode bits are
// present, so its presence is exactly the condition that can grant another
// account access past a 0600 mode.
const linuxPOSIXACLAccessAttribute = "system.posix_acl_access"

var (
	linuxFgetxattr    = unix.Fgetxattr
	linuxFremovexattr = unix.Fremovexattr
)

func certificateExportFileHasACL(file *os.File) (bool, error) {
	size, err := linuxFgetxattr(int(file.Fd()), linuxPOSIXACLAccessAttribute, nil)
	if err != nil {
		if errors.Is(err, unix.ENODATA) {
			return false, nil
		}
		// Unsupported xattrs do not prove that no ACL can grant access, so
		// protected inputs and outputs must fail closed on such filesystems.
		return false, fmt.Errorf("inspect access control list: %w", err)
	}
	return size > 0, nil
}

func clearCertificateExportFileACL(file *os.File) error {
	err := linuxFremovexattr(int(file.Fd()), linuxPOSIXACLAccessAttribute)
	if err == nil || errors.Is(err, unix.ENODATA) {
		return nil
	}
	return fmt.Errorf("remove access control list: %w", err)
}
