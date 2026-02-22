//go:build !windows

package shared

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestWriteFileNoSymlinkOverwrite_RespectsUmask(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "profile.mobileprovision")

	oldUmask := syscall.Umask(0o077)
	defer syscall.Umask(oldUmask)

	_, err := WriteFileNoSymlinkOverwrite(outPath, bytes.NewReader([]byte("hello")), 0o644, ".asc-profile-*", ".asc-profile-backup-*")
	if err != nil {
		t.Fatalf("WriteFileNoSymlinkOverwrite error: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}

	// 0644 masked by 0077 => 0600.
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("mode = %o, want %o", got, want)
	}
}
