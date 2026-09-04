//go:build darwin || linux

package screenshots

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func openMatrixReviewLockFile(root *os.Root, name string) (*os.File, error) {
	file, err := root.OpenFile(name, os.O_CREATE|os.O_RDWR|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func tryMatrixReviewFileLock(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
		return false, nil
	}
	return false, err
}

func unlockMatrixReviewFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
