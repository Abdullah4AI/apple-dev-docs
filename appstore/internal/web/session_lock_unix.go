//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package web

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func acquireSharedSessionLockFile(path string) (func(), bool) {
	dir := filepath.Dir(path)
	if !prepareSharedSessionLockDir(dir) {
		return nil, false
	}
	return acquirePreparedLockFile(path, openSharedSessionLockFile)
}

func prepareSharedSessionLockDir(dir string) bool {
	parent := filepath.Dir(dir)
	var parentStat unix.Stat_t
	if err := unix.Lstat(parent, &parentStat); err != nil ||
		parentStat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		parentStat.Uid != uint32(os.Getuid()) ||
		parentStat.Mode&0o022 != 0 {
		return false
	}

	if err := os.Mkdir(dir, 0o700); err != nil && !os.IsExist(err) {
		return false
	}
	var dirStat unix.Stat_t
	if err := unix.Lstat(dir, &dirStat); err != nil ||
		dirStat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		dirStat.Uid != uint32(os.Getuid()) ||
		dirStat.Mode&0o777 != 0o700 {
		return false
	}
	return true
}

func openSharedSessionLockFile(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Uid != uint32(os.Getuid()) ||
		stat.Mode&0o777 != 0o600 {
		_ = unix.Close(fd)
		return nil, os.ErrPermission
	}
	return os.NewFile(uintptr(fd), path), nil
}

func lockSessionFile(file *os.File) error {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return errSessionLockHeld
	}
	return err
}

func unlockSessionFile(file *os.File) error { return unix.Flock(int(file.Fd()), unix.LOCK_UN) }
