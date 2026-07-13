//go:build darwin || linux

package telemetry

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func tryLockStateFile(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return false, err
}

func unlockStateFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

func openStateFileForRead(path string) (*os.File, error) {
	return openTelemetryFileForRead(path)
}

func openTelemetryFileForRead(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NONBLOCK|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, &os.PathError{Op: "open", Path: path, Err: unix.EBADF}
	}
	return file, nil
}

func openStateLockForStat(path string) (*os.File, error) {
	return os.Open(path)
}

func replaceStateFile(oldPath, newPath string, _ time.Duration) error {
	return replaceTelemetryFile(oldPath, newPath, 0)
}

func replaceTelemetryFile(oldPath, newPath string, _ time.Duration) error {
	return os.Rename(oldPath, newPath)
}

func syncTelemetryDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil && !errors.Is(err, unix.EINVAL) && !errors.Is(err, unix.ENOTSUP) {
		return err
	}
	return nil
}

func removeLegacyStateLockDirectory(path string) error {
	return unix.Rmdir(path)
}
