//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly

package shared

import (
	"os"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

// OpenNewFileNoFollow creates a new file without following symlinks.
// Uses a best-effort pre/post open validation sequence because a portable
// O_NOFOLLOW equivalent is not available on this platform.
func OpenNewFileNoFollow(path string, perm os.FileMode) (*os.File, error) {
	return secureopen.OpenNewFileNoFollow(path, perm)
}

// OpenExistingNoFollow opens an existing file with best-effort symlink checks.
// The validation/open sequence reduces, but does not eliminate, TOCTOU risk.
func OpenExistingNoFollow(path string) (*os.File, error) {
	return secureopen.OpenExistingNoFollow(path)
}
