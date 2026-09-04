//go:build !darwin && !linux && !windows

package screenshots

import (
	"errors"
	"os"
)

func openMatrixReviewLockFile(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_CREATE|os.O_RDWR, 0o600)
}

func tryMatrixReviewFileLock(*os.File) (bool, error) {
	return false, errors.New("matrix review locking is unsupported on this platform")
}

func unlockMatrixReviewFile(*os.File) error { return nil }
