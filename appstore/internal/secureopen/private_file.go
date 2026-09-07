package secureopen

import (
	"fmt"
	"os"
)

// OpenNewPrivateFileNoFollowInRoot creates a new owner-only file beneath a
// rooted directory. The platform implementation establishes the strongest
// private-file policy available before the caller can write secret bytes.
//
// The caller remains responsible for publishing the returned file through an
// atomic, no-follow operation.
func OpenNewPrivateFileNoFollowInRoot(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	if perm.Perm()&0o077 != 0 {
		return nil, fmt.Errorf("private file mode %#o is not owner-only", perm.Perm())
	}
	return openNewPrivateFileNoFollowInRoot(root, name, perm)
}

// PreparePrivateFile applies and verifies the owner-only policy to an open
// private file. It must run before any secret bytes are written.
func PreparePrivateFile(file *os.File, perm os.FileMode) error {
	if perm.Perm()&0o077 != 0 {
		return fmt.Errorf("private file mode %#o is not owner-only", perm.Perm())
	}
	return preparePrivateFile(file, perm)
}
