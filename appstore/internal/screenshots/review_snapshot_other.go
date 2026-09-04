//go:build !windows

package screenshots

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func createMatrixReviewSnapshotDir() (string, error) {
	return os.MkdirTemp("", matrixReviewSnapshotDirPrefix+"*")
}

// openMatrixReviewSnapshotDirInRoot creates and opens a snapshot subdirectory
// through the already-pinned snapshot root. The returned root is owned by the
// caller and must be closed independently.
func openMatrixReviewSnapshotDirInRoot(root *os.Root, relative string) (*os.Root, error) {
	if root == nil {
		return nil, errors.New("snapshot root is unavailable")
	}
	if relative == "" || relative == "." {
		return root.OpenRoot(".")
	}
	if filepath.IsAbs(relative) {
		return nil, errors.New("snapshot directory escapes root")
	}
	clean := filepath.Clean(relative)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, errors.New("snapshot directory escapes root")
	}
	if err := root.MkdirAll(clean, 0o700); err != nil {
		return nil, err
	}
	info, err := root.Lstat(clean)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("snapshot asset directory is not a real directory")
	}
	return root.OpenRoot(clean)
}

func matrixReviewSnapshotDirIsProtected(info os.FileInfo, _ string) bool {
	return info != nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o700
}

func createMatrixReviewSnapshotFileInRoot(root *os.Root, name, _ string) (*os.File, error) {
	if root == nil {
		return nil, errors.New("snapshot root is unavailable")
	}
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return nil, errors.New("snapshot file name is invalid")
	}
	return root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
}
