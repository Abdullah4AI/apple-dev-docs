//go:build linux

package secureopen

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// RenameNoReplaceInRoot atomically renames oldName to newName beneath root and
// fails when newName already exists.
func RenameNoReplaceInRoot(root *os.Root, oldName, newName string) error {
	const op = "renameat2"
	if err := validateRenameNoReplaceNames(oldName, newName); err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}

	directory, err := root.Open(".")
	if err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}
	defer directory.Close()
	raw, err := directory.SyscallConn()
	if err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}
	var renameErr error
	if err := raw.Control(func(fd uintptr) {
		renameErr = unix.Renameat2(int(fd), oldName, int(fd), newName, unix.RENAME_NOREPLACE)
	}); err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}
	if errors.Is(renameErr, unix.ENOSYS) || errors.Is(renameErr, unix.EINVAL) || errors.Is(renameErr, unix.EOPNOTSUPP) {
		return unsupportedRenameNoReplaceError(op, oldName, newName, renameErr)
	}
	return renameNoReplaceError(op, oldName, newName, renameErr)
}
