//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package distribute

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func openDistributionRunLockFile(runRoot *os.Root, name string) (*os.File, error) {
	file, err := runRoot.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR|unix.O_NOFOLLOW, 0o600)
	if err == nil {
		// os.OpenFile applies the process umask. Normalize the newly created,
		// exclusively owned inode before any other process may accept it.
		if chmodErr := file.Chmod(0o600); chmodErr != nil {
			_ = file.Close()
			return nil, chmodErr
		}
		return file, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	return runRoot.OpenFile(name, os.O_RDWR|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
}

func tryDistributionRunFileLock(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return false, err
}

func unlockDistributionRunFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

func validateDistributionRunLockPlatform(*os.File) error { return nil }
