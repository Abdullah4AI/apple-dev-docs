//go:build windows

package screenshots

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func createMatrixReviewSnapshotDir() (string, error) {
	return createMatrixOwnerOnlyTempDir(matrixReviewSnapshotDirPrefix)
}

// openMatrixReviewSnapshotDirInRoot walks snapshot subdirectories through the
// held root. New Windows directories receive the owner-only DACL at creation;
// an existing reparse point is rejected before it can be adopted.
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
	current, err := root.OpenRoot(".")
	if err != nil {
		return nil, err
	}
	closeCurrent := true
	defer func() {
		if closeCurrent {
			_ = current.Close()
		}
	}()
	for _, component := range strings.Split(filepath.ToSlash(clean), "/") {
		if component == "" || component == "." || component == ".." {
			return nil, errors.New("snapshot directory component is invalid")
		}
		info, lstatErr := current.Lstat(component)
		if errors.Is(lstatErr, os.ErrNotExist) {
			if err := createMatrixOwnerOnlyDirectoryInRoot(current, component); err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) && !errors.Is(err, windows.ERROR_FILE_EXISTS) {
				return nil, err
			}
			info, lstatErr = current.Lstat(component)
		}
		if lstatErr != nil {
			return nil, lstatErr
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("snapshot asset directory is not a real directory")
		}
		next, err := current.OpenRoot(component)
		if err != nil {
			return nil, err
		}
		after, err := next.Stat(".")
		if err != nil || !os.SameFile(info, after) {
			_ = next.Close()
			if err != nil {
				return nil, err
			}
			return nil, errors.New("snapshot asset directory changed while opening")
		}
		_ = current.Close()
		current = next
	}
	closeCurrent = false
	return current, nil
}

func createMatrixReviewSnapshotFileInRoot(root *os.Root, name, displayPath string) (*os.File, error) {
	return createMatrixOwnerOnlyFileInRoot(root, name, displayPath)
}

func matrixReviewSnapshotDirIsProtected(info os.FileInfo, path string) bool {
	if info == nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || strings.TrimSpace(path) == "" {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	return matrixOwnerOnlyProtectedDACL(file)
}
