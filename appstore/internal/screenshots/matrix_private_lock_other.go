//go:build !windows

package screenshots

import (
	"os"
	"path/filepath"
)

func createMatrixPrivateScratchDir(prefix string) (string, error) {
	return os.MkdirTemp("", prefix)
}

func createMatrixPrivateAttemptParent() (string, error) {
	namespace, err := createMatrixPrivateScratchDir(".asc-matrix-attempt-ns-")
	if err != nil {
		return "", err
	}
	parent := filepath.Join(namespace, "parent")
	if err := os.Mkdir(parent, 0o700); err != nil {
		_ = os.RemoveAll(namespace)
		return "", err
	}
	return parent, nil
}

func createMatrixPrivateAttemptChild(parent *os.Root, _ string, name string) error {
	return parent.Mkdir(name, 0o700)
}

func createMatrixPrivateAttemptOutputDirInRoot(parent *os.Root) error {
	return parent.Mkdir("output", 0o700)
}

func createMatrixPrivateAttemptOutputDir(workDir string) error {
	return os.MkdirAll(filepath.Join(workDir, "output"), 0o755)
}

func createMatrixPrivateAttemptFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
}

func createMatrixPrivateAttemptFileInRoot(parent *os.Root, name, _ string) (*os.File, error) {
	return parent.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
}

func lockMatrixPrivateAttemptDirectory(root *os.Root) error {
	return root.Chmod(".", 0o500)
}

func unlockMatrixPrivateAttemptDirectory(root *os.Root) error {
	return root.Chmod(".", 0o700)
}

// lockMatrixPrivateAttemptFile makes a path handed to an external adapter
// readable but not writable. The directory containing the file is separately
// locked while the adapter runs, so the path cannot be replaced either.
func lockMatrixPrivateAttemptFile(path string) error {
	return os.Chmod(path, 0o400)
}

func unlockMatrixPrivateAttemptFile(path string) error {
	return os.Chmod(path, 0o600)
}

func lockMatrixPrivateAttemptFileHandle(file *os.File) error {
	if file == nil {
		return os.ErrInvalid
	}
	return file.Chmod(0o400)
}
