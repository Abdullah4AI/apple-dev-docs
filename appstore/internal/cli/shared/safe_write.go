package shared

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

// Publication primitives are indirected so tests can simulate destination
// filesystems that do not implement them.
var (
	renameNoReplaceInRoot = secureopen.RenameNoReplaceInRoot
	linkInRoot            = func(root *os.Root, oldName, newName string) error {
		return root.Link(oldName, newName)
	}
)

// SafeWriteFileNoSymlink writes a file to path without following symlinks and with an optional
// overwrite mode that preserves the original destination until the new file is fully written.
//
// When overwrite is false, the destination must not already exist.
// When overwrite is true, we refuse to overwrite symlinks and we use temp+rename; if rename fails
// because the destination exists (notably on Windows), we fall back to a safe replace that uses a
// backup file to preserve the original if the final move fails.
func SafeWriteFileNoSymlink(path string, perm os.FileMode, overwrite bool, tempPattern string, backupPattern string, write func(*os.File) (int64, error)) (int64, error) {
	if len(path) > 0 && os.IsPathSeparator(path[len(path)-1]) {
		return 0, fmt.Errorf("output path %q must be a file", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}

	if !overwrite {
		// Callers pass a complete, already-resolved output path. Pin its immediate
		// parent so staging, publication, and cleanup cannot be redirected by a
		// concurrent parent rename.
		parent, err := os.OpenRoot(filepath.Dir(path))
		if err != nil {
			return 0, err
		}
		defer parent.Close()

		base := filepath.Base(path)
		if _, err := parent.Lstat(base); err == nil {
			return 0, existingOutputError(path, os.ErrExist)
		} else if !errors.Is(err, os.ErrNotExist) {
			return 0, err
		}

		file, temporaryName, err := secureopen.CreateTempNoFollowInRoot(parent, ".", tempPattern, perm)
		if err != nil {
			return 0, fmt.Errorf("create output %q: %w", path, replaceErrorPaths(err, path, temporaryName))
		}
		temporaryPath := file.Name()
		displayDestinationError := func(err error) error {
			return replaceErrorPaths(err, path, temporaryPath, temporaryName)
		}
		displayTemporaryError := func(err error) error {
			return replaceErrorPaths(err, "temporary output", temporaryPath, temporaryName)
		}
		displayPublishError := func(err error) error {
			if err == nil {
				return nil
			}
			err = displayDestinationError(err)
			if errors.Is(err, os.ErrExist) {
				return existingOutputError(path, err)
			}
			// Publication primitives report failures as *os.LinkError naming the
			// staged and destination entries relative to the pinned parent, which
			// reads as the same file twice once the staged operand is rewritten to
			// the destination. Name the output path once and keep the syscall
			// error as the wrapped cause.
			var linkErr *os.LinkError
			if errors.As(err, &linkErr) {
				return &displayPathError{err: err, message: fmt.Sprintf("publish output %q: %v", path, linkErr.Err)}
			}
			return err
		}
		written, err := writeNewFileNoSymlink(temporaryName, file, func(file *os.File) (int64, error) {
			written, err := write(file)
			return written, displayDestinationError(err)
		}, newFileWriteOps{
			syncFile: func() error {
				return displayDestinationError(file.Sync())
			},
			closeFile: func() error {
				return displayDestinationError(file.Close())
			},
			removeFile: func(name string) error {
				return displayTemporaryError(removeRootedFile(parent, name))
			},
		})
		if err != nil {
			return written, err
		}

		if err := publishStagedFileNoReplace(temporaryName, base, stagedFilePublishOps{
			renameFile: func(oldName, newName string) error {
				return displayPublishError(renameNoReplaceInRoot(parent, oldName, newName))
			},
			linkFile: func(oldName, newName string) error {
				return displayPublishError(linkInRoot(parent, oldName, newName))
			},
			copyFile: func(oldName, newName string) error {
				return displayPublishError(copyStagedFileNoReplace(parent, oldName, newName, perm))
			},
			removeFile: func(name string) error {
				return displayTemporaryError(removeRootedFile(parent, name))
			},
		}); err != nil {
			return written, err
		}
		return written, nil
	}

	return writeFileNoSymlinkOverwrite(path, perm, tempPattern, backupPattern, write)
}

type newFileWriteOps struct {
	syncFile   func() error
	closeFile  func() error
	removeFile func(string) error
}

type stagedFilePublishOps struct {
	renameFile func(string, string) error
	linkFile   func(string, string) error
	copyFile   func(string, string) error
	removeFile func(string) error
}

func publishStagedFileNoReplace(temporaryName, destinationName string, ops stagedFilePublishOps) error {
	removed := false
	removeStaged := func() error {
		err := ops.removeFile(temporaryName)
		if err == nil {
			removed = true
		}
		return err
	}
	defer func() {
		if !removed {
			_ = removeStaged()
		}
	}()
	failAfterCleanup := func(err error) error {
		return cleanupIncompleteFile(temporaryName, err, func() error { return nil }, func(string) error {
			return removeStaged()
		})
	}

	renameErr := ops.renameFile(temporaryName, destinationName)
	if renameErr == nil {
		// Rename consumes the staged name, so no cleanup remains.
		removed = true
		return nil
	}
	if !errors.Is(renameErr, secureopen.ErrRenameNoReplaceUnsupported) {
		return failAfterCleanup(renameErr)
	}

	// Filesystems without a native no-replace rename may still support hard
	// links. Link is also atomic and cannot replace an existing destination.
	linkErr := ops.linkFile(temporaryName, destinationName)
	if linkErr == nil {
		// Once publication succeeds, the complete destination is committed. Cleanup
		// remains best effort so callers do not retry a write that already succeeded.
		_ = removeStaged()
		return nil
	}
	if errors.Is(linkErr, os.ErrExist) {
		return failAfterCleanup(linkErr)
	}

	// FAT/exFAT volumes, SMB shares, and several FUSE mounts implement neither
	// primitive, and they report that with filesystem-specific errors rather
	// than one portable code. Copying into an exclusively created destination is
	// the no-clobber publication every filesystem supports, so try it before
	// discarding a complete download.
	if err := ops.copyFile(temporaryName, destinationName); err != nil {
		return failAfterCleanup(err)
	}
	_ = removeStaged()
	return nil
}

// copyStagedFileNoReplace publishes the staged file by creating the destination
// with an exclusive, no-follow create and copying the staged bytes into it.
//
// Exclusive create cannot clobber an existing destination, so this preserves
// no-replace semantics on filesystems that support neither an atomic no-replace
// rename nor hard links. A partially copied destination is removed so callers
// never observe a truncated output file.
func copyStagedFileNoReplace(root *os.Root, temporaryName, destinationName string, perm os.FileMode) error {
	source, err := secureopen.OpenExistingNoFollowInRoot(root, temporaryName)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := secureopen.OpenNewFileNoFollowInRoot(root, destinationName, perm)
	if err != nil {
		return err
	}
	if err := copyIntoPublishedFile(destination, source); err != nil {
		return errors.Join(err, removeRootedFile(root, destinationName))
	}
	return nil
}

func copyIntoPublishedFile(destination, source *os.File) error {
	if _, err := io.Copy(destination, source); err != nil {
		return errors.Join(err, destination.Close())
	}
	if err := destination.Sync(); err != nil {
		return errors.Join(err, destination.Close())
	}
	return destination.Close()
}

func writeNewFileNoSymlink(path string, file *os.File, write func(*os.File) (int64, error), ops newFileWriteOps) (int64, error) {
	closed := false
	closeFile := func() error {
		if closed {
			return nil
		}
		closed = true
		return ops.closeFile()
	}
	defer func() {
		_ = closeFile()
	}()

	written, err := write(file)
	if err != nil {
		return written, cleanupIncompleteFile(path, err, closeFile, ops.removeFile)
	}
	if err := ops.syncFile(); err != nil {
		return written, cleanupIncompleteFile(path, err, closeFile, ops.removeFile)
	}
	if err := closeFile(); err != nil {
		return written, cleanupIncompleteFile(path, err, closeFile, ops.removeFile)
	}
	return written, nil
}

type displayPathError struct {
	err     error
	message string
}

func (e *displayPathError) Error() string {
	return e.message
}

func (e *displayPathError) Unwrap() error {
	return e.err
}

func replaceErrorPaths(err error, replacement string, paths ...string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	for _, path := range paths {
		if path != "" && path != replacement {
			message = strings.ReplaceAll(message, path, replacement)
		}
	}
	if message == err.Error() {
		return err
	}
	return &displayPathError{err: err, message: message}
}

// existingOutputError reports a refused overwrite against the output path the
// caller asked for while keeping the cause matchable with errors.Is.
func existingOutputError(path string, cause error) error {
	return &displayPathError{err: cause, message: fmt.Sprintf("output file already exists: %q", path)}
}

func removeRootedFile(root *os.Root, name string) error {
	err := root.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func cleanupIncompleteFile(path string, primaryErr error, closeFile func() error, removeFile func(string) error) error {
	closeErr := closeFile()
	removeErr := removeFile(path)
	if closeErr == nil && removeErr == nil {
		return primaryErr
	}
	return errors.Join(primaryErr, closeErr, removeErr)
}
