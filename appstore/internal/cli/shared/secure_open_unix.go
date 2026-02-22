//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package shared

import (
	"os"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

// OpenNewFileNoFollow creates a new file without following symlinks.
// Uses O_EXCL to prevent overwriting existing files and O_NOFOLLOW to prevent symlink attacks.
func OpenNewFileNoFollow(path string, perm os.FileMode) (*os.File, error) {
	return secureopen.OpenNewFileNoFollow(path, perm)
}

// OpenExistingNoFollow opens an existing file without following symlinks.
func OpenExistingNoFollow(path string) (*os.File, error) {
	return secureopen.OpenExistingNoFollow(path)
}
