//go:build linux

package notarization

import (
	"os"

	"golang.org/x/sys/unix"
)

func openStaplerSearchableDirectoryNoFollow(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_PATH|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	if err := unix.Faccessat(fd, ".", unix.X_OK, unix.AT_EACCESS); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}
