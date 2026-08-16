//go:build windows

package rootfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestUnsupportedDirectorySyncErrorOnWindows(t *testing.T) {
	if !unsupportedDirectorySyncError(windows.ERROR_ACCESS_DENIED) {
		t.Fatal("ERROR_ACCESS_DENIED should be treated as an unsupported directory sync")
	}
	if unsupportedDirectorySyncError(windows.ERROR_WRITE_FAULT) {
		t.Fatal("ERROR_WRITE_FAULT should remain a directory sync failure")
	}
}

func TestCreateNewFileAtomicPublishesOnWindows(t *testing.T) {
	dir := t.TempDir()
	root := mustRoot(t, dir)
	if err := root.CreateNewFileAtomic("plan.json", []byte("planned"), 0o600); err != nil {
		t.Fatalf("CreateNewFileAtomic() error = %v", err)
	}
	if got := mustRead(t, filepath.Join(dir, "plan.json")); got != "planned" {
		t.Fatalf("published content = %q, want planned", got)
	}
	if err := root.CreateNewFileAtomic("plan.json", []byte("replacement"), 0o600); !errors.Is(err, os.ErrExist) {
		t.Fatalf("retry error = %v, want os.ErrExist", err)
	}
}
