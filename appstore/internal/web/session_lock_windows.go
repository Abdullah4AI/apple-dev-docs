//go:build windows

package web

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func acquireSharedSessionLockFile(path string) (func(), bool) { return acquireLockFile(path) }

func lockSessionFile(file *os.File) error {
	var overlapped windows.Overlapped
	err := windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlapped)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
		return errSessionLockHeld
	}
	return err
}

func unlockSessionFile(file *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlapped)
}
