package screenshots

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
)

const (
	matrixReviewLockName         = ".asc-matrix-review.lock"
	matrixReviewLockPollInterval = 25 * time.Millisecond
)

// acquireMatrixReviewLock serializes the complete HTML-plus-manifest
// transaction for one review directory. The lock inode remains in place so a
// second process can never lock a replacement inode while a first process
// still owns the original.
func acquireMatrixReviewLock(ctx context.Context, root rootfs.Root) (func() error, error) {
	return acquireMatrixNamedLock(ctx, root, matrixReviewLockName)
}

func acquireMatrixNamedLock(ctx context.Context, root rootfs.Root, name string) (func() error, error) {
	if ctx == nil {
		return nil, errors.New("matrix lock context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rooted, err := root.OpenRoot()
	if err != nil {
		return nil, err
	}
	closeRoot := true
	defer func() {
		if closeRoot {
			_ = rooted.Close()
		}
	}()
	file, err := openMatrixReviewLockFile(rooted, name)
	if err != nil {
		return nil, err
	}
	closeFile := true
	defer func() {
		if closeFile {
			_ = file.Close()
		}
	}()
	if err := validateMatrixLockPath(rooted, file, name); err != nil {
		return nil, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		locked, lockErr := tryMatrixReviewFileLock(file)
		if lockErr != nil {
			return nil, lockErr
		}
		if locked {
			if err := validateMatrixLockPath(rooted, file, name); err != nil {
				return nil, errors.Join(err, unlockMatrixReviewFile(file))
			}
			closeRoot = false
			closeFile = false
			var once sync.Once
			var releaseErr error
			return func() error {
				once.Do(func() {
					releaseErr = errors.Join(unlockMatrixReviewFile(file), file.Close(), rooted.Close())
				})
				return releaseErr
			}, nil
		}

		timer := time.NewTimer(matrixReviewLockPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func validateMatrixLockPath(root *os.Root, file *os.File, name string) error {
	openedInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect matrix lock: %w", err)
	}
	if !openedInfo.Mode().IsRegular() {
		return errors.New("matrix lock must be a regular file")
	}
	pathInfo, err := root.Lstat(name)
	if err != nil {
		return fmt.Errorf("inspect matrix lock path: %w", err)
	}
	if !pathInfo.Mode().IsRegular() || !os.SameFile(openedInfo, pathInfo) {
		return errors.New("matrix lock path changed")
	}
	return nil
}
