//go:build darwin

package notarization

import (
	"os"

	"golang.org/x/sys/unix"
)

// Darwin exposes O_SEARCH as O_EXEC|O_DIRECTORY, but x/sys/unix does not
// currently export the composite constant. A search-only open checks the
// permission needed for pathname traversal without also requiring directory
// read permission.
const darwinOSearch = 0x40000000 | unix.O_DIRECTORY

func openStaplerSearchableDirectoryNoFollow(path string) (*os.File, error) {
	fd, err := unix.Open(path, darwinOSearch|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}
