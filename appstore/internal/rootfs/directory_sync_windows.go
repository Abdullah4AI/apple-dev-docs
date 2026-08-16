//go:build windows

package rootfs

import (
	"errors"

	"golang.org/x/sys/windows"
)

func unsupportedDirectorySyncError(err error) bool {
	// os.File.Sync delegates to FlushFileBuffers. Read-only directory handles
	// commonly reject that unsupported durability operation after publication.
	return errors.Is(err, windows.ERROR_ACCESS_DENIED) ||
		errors.Is(err, windows.ERROR_INVALID_FUNCTION) ||
		errors.Is(err, windows.ERROR_NOT_SUPPORTED)
}
