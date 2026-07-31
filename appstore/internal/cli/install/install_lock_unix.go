//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package install

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

var errInstallLockHeld = errors.New("install lock is held")

func lockInstallFile(file *os.File) error {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return errInstallLockHeld
	}
	return err
}

func unlockInstallFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
