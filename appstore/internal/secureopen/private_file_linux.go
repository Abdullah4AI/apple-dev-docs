//go:build linux

package secureopen

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

const linuxPrivateFileACLAttribute = "system.posix_acl_access"

var (
	linuxPrivateFileFgetxattr    = unix.Fgetxattr
	linuxPrivateFileFremovexattr = unix.Fremovexattr
)

func openNewPrivateFileNoFollowInRoot(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	return OpenNewFileNoFollowInRoot(root, name, perm)
}

func preparePrivateFile(file *os.File, perm os.FileMode) error {
	if file == nil {
		return fmt.Errorf("private file is nil")
	}
	if perm.Perm()&0o077 != 0 {
		return fmt.Errorf("private file mode %#o is not owner-only", perm.Perm())
	}
	if err := file.Chmod(perm.Perm()); err != nil {
		return fmt.Errorf("set private file permissions: %w", err)
	}
	if err := clearLinuxPrivateFileACL(file); err != nil {
		return fmt.Errorf("remove private file ACL: %w", err)
	}
	hasACL, err := linuxPrivateFileHasACL(file)
	if err != nil {
		return fmt.Errorf("verify private file ACL: %w", err)
	}
	if hasACL {
		return fmt.Errorf("private file retains an extended ACL")
	}
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("verify private file permissions: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("private file permissions %#o are not owner-only", info.Mode().Perm())
	}
	return nil
}

func linuxPrivateFileHasACL(file *os.File) (bool, error) {
	size, err := linuxPrivateFileFgetxattr(int(file.Fd()), linuxPrivateFileACLAttribute, nil)
	if err != nil {
		if errors.Is(err, unix.ENODATA) {
			return false, nil
		}
		// Unsupported xattrs do not prove that no named ACL grants another
		// account access, so the secret-file path fails closed.
		return false, err
	}
	return size > 0, nil
}

func clearLinuxPrivateFileACL(file *os.File) error {
	err := linuxPrivateFileFremovexattr(int(file.Fd()), linuxPrivateFileACLAttribute)
	if err == nil || errors.Is(err, unix.ENODATA) {
		return nil
	}
	return err
}
