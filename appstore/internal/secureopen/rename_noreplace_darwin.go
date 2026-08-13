//go:build darwin

package secureopen

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// RenameNoReplaceInRoot atomically renames oldName to newName beneath root and
// fails when newName already exists.
func RenameNoReplaceInRoot(root *os.Root, oldName, newName string) error {
	const op = "renameatx_np"
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
		renameErr = unix.RenameatxNp(int(fd), oldName, int(fd), newName, unix.RENAME_EXCL)
	}); err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}
	if renameNoReplaceUnsupportedDarwin(renameErr) {
		return unsupportedRenameNoReplaceError(op, oldName, newName, renameErr)
	}
	return renameNoReplaceError(op, oldName, newName, renameErr)
}

func renameNoReplaceUnsupportedDarwin(err error) bool {
	return errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.ENOSYS)
}
