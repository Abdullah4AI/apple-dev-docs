//go:build darwin

package notarization

import (
	"errors"
	"os"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"golang.org/x/sys/unix"
)

func newStaplerRegularFileAccess(absolute, workingDirectoryPath string, workingDirectory *os.File) (*staplerRegularFileAccess, error) {
	return newStaplerRegularFileAccessWithOps(absolute, workingDirectoryPath, workingDirectory, staplerDarwinRegularFileOpenOps())
}

func staplerDarwinRegularFileOpenOps() staplerRegularFileOpenOps {
	return staplerRegularFileOpenOps{
		openSearchAt: func(parent *os.File, name string) (*os.File, error) {
			return openStaplerDarwinAt(parent, name, darwinOSearch|unix.O_NOFOLLOW|unix.O_CLOEXEC)
		},
		openFinalAt: func(parent *os.File, name string) (*os.File, error) {
			return openStaplerDarwinAt(parent, name, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC)
		},
	}
}

func openStaplerDarwinAt(parent *os.File, name string, flags int) (*os.File, error) {
	if parent == nil {
		return nil, errors.New("regular-file parent descriptor is missing")
	}
	connection, err := parent.SyscallConn()
	if err != nil {
		return nil, err
	}
	var (
		fd      int
		openErr error
	)
	if err := connection.Control(func(rawFD uintptr) {
		fd, openErr = unix.Openat(int(rawFD), name, flags, 0)
	}); err != nil {
		return nil, err
	}
	if openErr != nil {
		if errors.Is(openErr, unix.ELOOP) {
			return nil, rootfs.ErrSymlink
		}
		return nil, openErr
	}
	return os.NewFile(uintptr(fd), name), nil
}
