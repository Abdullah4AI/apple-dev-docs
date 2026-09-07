//go:build !windows && !darwin && !linux

package secureopen

import (
	"fmt"
	"os"
)

func openNewPrivateFileNoFollowInRoot(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	return OpenNewFileNoFollowInRoot(root, name, perm)
}

func preparePrivateFile(file *os.File, perm os.FileMode) error {
	if file == nil {
		return fmt.Errorf("private file is nil")
	}
	if perm.Perm()&0o077 != 0 {
		return fmt.Errorf("private file mode %#o is not owner-only", perm.Perm())
	}
	if err := file.Chmod(perm.Perm()); err != nil {
		return fmt.Errorf("set private file permissions: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("verify private file permissions: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("private file permissions %#o are not owner-only", info.Mode().Perm())
	}
	return nil
}
