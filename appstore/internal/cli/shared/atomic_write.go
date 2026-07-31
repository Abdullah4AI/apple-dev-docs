package shared

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

func createTempFileNoFollowWithPerm(dir string, pattern string, perm os.FileMode) (*os.File, error) {
	return secureopen.CreateTempNoFollow(dir, pattern, perm)
}

func writeFileNoSymlinkOverwrite(path string, perm os.FileMode, tempPattern string, backupPattern string, write func(*os.File) (int64, error)) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}

	// Do not remove/replace a symlink.
	hadExisting := false
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return 0, fmt.Errorf("refusing to overwrite symlink %q", path)
		}
		if info.IsDir() {
			return 0, fmt.Errorf("output path %q is a directory", path)
		}
		hadExisting = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}

	tempFile, err := createTempFileNoFollowWithPerm(filepath.Dir(path), tempPattern, perm)
	if err != nil {
		return 0, err
	}
	defer tempFile.Close()

	tempPath := tempFile.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	// Ensure final file permissions match caller intent rather than process umask.
	if err := tempFile.Chmod(perm); err != nil {
		return 0, err
	}

	written, err := write(tempFile)
	if err != nil {
		return 0, err
	}
	if err := tempFile.Sync(); err != nil {
		return 0, err
	}
	if err := tempFile.Close(); err != nil {
		return 0, err
	}

	// On Unix, rename replaces the destination atomically. On Windows, rename fails if the
	// destination exists, so we fall back to a safe replace that preserves the original
	// file if the final move fails.
	if err := os.Rename(tempPath, path); err != nil {
		if !hadExisting {
			return 0, err
		}

		backupFile, backupErr := os.CreateTemp(filepath.Dir(path), backupPattern)
		if backupErr != nil {
			return 0, err
		}
		backupPath := backupFile.Name()
		if closeErr := backupFile.Close(); closeErr != nil {
			return 0, closeErr
		}
		if removeErr := os.Remove(backupPath); removeErr != nil {
			return 0, removeErr
		}

		if moveErr := os.Rename(path, backupPath); moveErr != nil {
			return 0, moveErr
		}
		if moveErr := os.Rename(tempPath, path); moveErr != nil {
			_ = os.Rename(backupPath, path)
			return 0, moveErr
		}
		_ = os.Remove(backupPath)
	}

	success = true
	return written, nil
}

// WriteFileNoSymlinkOverwrite writes reader to path via temp+rename.
// It refuses to overwrite symlinks and uses a Windows-safe replace when needed.
func WriteFileNoSymlinkOverwrite(path string, reader io.Reader, perm os.FileMode, tempPattern string, backupPattern string) (int64, error) {
	return writeFileNoSymlinkOverwrite(path, perm, tempPattern, backupPattern, func(file *os.File) (int64, error) {
		return io.Copy(file, reader)
	})
}
