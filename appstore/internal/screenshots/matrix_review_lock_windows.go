//go:build windows

package screenshots

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func openMatrixReviewLockFile(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_CREATE|os.O_RDWR, 0o600)
}

func tryMatrixReviewFileLock(file *os.File) (bool, error) {
	var overlapped windows.Overlapped
	err := windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		&overlapped,
	)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
		return false, nil
	}
	return false, err
}

func unlockMatrixReviewFile(file *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlapped)
}
