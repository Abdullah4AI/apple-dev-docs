//go:build !windows

package rootfs

import (
	"errors"
	"syscall"
)

func unsupportedDirectorySyncError(err error) bool {
	return errors.Is(err, syscall.EINVAL) ||
		errors.Is(err, syscall.ENOTSUP) ||
		errors.Is(err, syscall.ENOSYS)
}
