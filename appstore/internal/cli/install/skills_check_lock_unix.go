//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package install

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

var fcntlSkillsCheckClaim = unix.FcntlFlock

func lockSkillsCheckClaimFile(file *os.File, wait bool) (bool, error) {
	command := unix.F_SETLK
	if wait {
		command = unix.F_SETLKW
	}
	lock := unix.Flock_t{
		Type:   unix.F_WRLCK,
		Whence: 0,
		Start:  0,
		Len:    1,
	}
	if err := retrySkillsCheckFcntl(file.Fd(), command, &lock); err != nil {
		if !wait && (errors.Is(err, unix.EACCES) || errors.Is(err, unix.EAGAIN)) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func unlockSkillsCheckClaimFile(file *os.File) error {
	lock := unix.Flock_t{
		Type:   unix.F_UNLCK,
		Whence: 0,
		Start:  0,
		Len:    1,
	}
	return retrySkillsCheckFcntl(file.Fd(), unix.F_SETLK, &lock)
}

func retrySkillsCheckFcntl(fd uintptr, command int, lock *unix.Flock_t) error {
	for {
		err := fcntlSkillsCheckClaim(fd, command, lock)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}
