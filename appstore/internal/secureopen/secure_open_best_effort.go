package secureopen

import (
	"errors"
	"fmt"
	"os"
)

type (
	existingFileOpener func(path string) (*os.File, error)
	newFileCreator     func(path string, perm os.FileMode) (*os.File, error)
)

// openExistingNoFollowBestEffort provides a portable, best-effort "no-follow"
// implementation for platforms that do not expose O_NOFOLLOW.
//
// It validates that the path is not a symlink before open, then verifies that
// the opened file still matches the same path after open. This shrinks the
// TOCTOU window but cannot make the operation fully atomic.
func openExistingNoFollowBestEffort(path string, opener existingFileOpener) (*os.File, error) {
	before, err := lstatNoSymlink(path)
	if err != nil {
		return nil, err
	}

	file, err := opener(path)
	if err != nil {
		return nil, err
	}

	if err := verifyOpenedPath(path, file, before); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

// openWritableFileNoFollowBestEffort provides a portable, best-effort
// "no-follow" path for creating or appending to a file on platforms that do not
// expose O_NOFOLLOW.
//
// It rejects symlink paths before the open and verifies the resulting file
// descriptor still maps to the same path afterwards. This reduces, but cannot
// eliminate, TOCTOU risk on platforms without atomic no-follow open.
func openWritableFileNoFollowBestEffort(path string, perm os.FileMode, creator newFileCreator) (*os.File, error) {
	if _, err := lstatNoSymlink(path); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	file, err := creator(path, perm)
	if err != nil {
		return nil, err
	}

	if err := verifyOpenedPath(path, file, nil); err != nil {
		return nil, closeAfterVerificationFailure(file, err)
	}
	return file, nil
}

// closeAfterVerificationFailure closes the handle but deliberately leaves the
// pathname untouched. Once identity verification fails, the name may refer to
// a concurrent replacement rather than the file opened by this process.
func closeAfterVerificationFailure(file *os.File, verifyErr error) error {
	if closeErr := file.Close(); closeErr != nil {
		return errors.Join(verifyErr, closeErr)
	}
	return verifyErr
}

func lstatNoSymlink(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to follow symlink %q", path)
	}
	return info, nil
}

func verifyOpenedPath(path string, file *os.File, before os.FileInfo) error {
	openedInfo, err := file.Stat()
	if err != nil {
		return err
	}

	after, err := lstatNoSymlink(path)
	if err != nil {
		return err
	}

	if before != nil && !os.SameFile(before, after) {
		return fmt.Errorf("file changed during open %q", path)
	}
	if !os.SameFile(after, openedInfo) {
		return fmt.Errorf("file changed during open %q", path)
	}
	return nil
}

func openExistingNoFollowInRootBestEffort(root *os.Root, name string, opener func() (*os.File, error)) (*os.File, error) {
	before, err := rootLstatNoSymlink(root, name)
	if err != nil {
		return nil, err
	}
	file, err := opener()
	if err != nil {
		return nil, err
	}
	if err := verifyRootOpenedPath(root, name, file, before); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

// openWritableFileNoFollowInRootBestEffort applies the same best-effort
// no-follow checks as openWritableFileNoFollowBestEffort to a root-relative
// name, for creating or appending to a file beneath root.
func openWritableFileNoFollowInRootBestEffort(root *os.Root, name string, opener func() (*os.File, error)) (*os.File, error) {
	if _, err := rootLstatNoSymlink(root, name); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	file, err := opener()
	if err != nil {
		return nil, err
	}
	if err := verifyRootOpenedPath(root, name, file, nil); err != nil {
		return nil, closeAfterVerificationFailure(file, err)
	}
	return file, nil
}

func rootLstatNoSymlink(root *os.Root, name string) (os.FileInfo, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to follow symlink %q", name)
	}
	return info, nil
}

func verifyRootOpenedPath(root *os.Root, name string, file *os.File, before os.FileInfo) error {
	openedInfo, err := file.Stat()
	if err != nil {
		return err
	}
	after, err := rootLstatNoSymlink(root, name)
	if err != nil {
		return err
	}
	if before != nil && !os.SameFile(before, after) {
		return fmt.Errorf("file changed during open %q", name)
	}
	if !os.SameFile(after, openedInfo) {
		return fmt.Errorf("file changed during open %q", name)
	}
	return nil
}
