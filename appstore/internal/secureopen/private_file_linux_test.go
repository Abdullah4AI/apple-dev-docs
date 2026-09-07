//go:build linux

package secureopen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestPreparePrivateFileFailsClosedWhenACLInspectionIsUnsupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-artifact")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer file.Close()

	previous := linuxPrivateFileFgetxattr
	t.Cleanup(func() { linuxPrivateFileFgetxattr = previous })
	linuxPrivateFileFgetxattr = func(int, string, []byte) (int, error) {
		return 0, unix.ENOTSUP
	}

	err = PreparePrivateFile(file, 0o600)
	if err == nil || !errors.Is(err, unix.ENOTSUP) {
		t.Fatalf("PreparePrivateFile() error = %v, want ENOTSUP", err)
	}
}

func TestPreparePrivateFileRemovesLinuxACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-artifact")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer file.Close()

	previousGet := linuxPrivateFileFgetxattr
	previousRemove := linuxPrivateFileFremovexattr
	t.Cleanup(func() {
		linuxPrivateFileFgetxattr = previousGet
		linuxPrivateFileFremovexattr = previousRemove
	})
	removed := false
	linuxPrivateFileFremovexattr = func(int, string) error {
		removed = true
		return nil
	}
	linuxPrivateFileFgetxattr = func(int, string, []byte) (int, error) {
		return 0, unix.ENODATA
	}

	if err := PreparePrivateFile(file, 0o600); err != nil {
		t.Fatalf("PreparePrivateFile() error = %v", err)
	}
	if !removed {
		t.Fatal("PreparePrivateFile() did not remove the ACL attribute")
	}
}
