package secureopen

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrivateFileCreatorAndPreparation(t *testing.T) {
	directory := t.TempDir()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	file, err := OpenNewPrivateFileNoFollowInRoot(root, ".private-artifact", 0o600)
	if err != nil {
		t.Fatalf("OpenNewPrivateFileNoFollowInRoot() error = %v", err)
	}
	defer file.Close()
	defer func() {
		_ = root.Remove(filepath.Base(file.Name()))
	}()

	if err := PreparePrivateFile(file, 0o600); err != nil {
		t.Fatalf("PreparePrivateFile() error = %v", err)
	}
	if _, err := file.WriteString("secret"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := file.Stat()
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		if got := info.Mode().Perm(); got&0o077 != 0 {
			t.Fatalf("private file permissions = %#o, want owner-only", got)
		}
	}
}

func TestPreparePrivateFileRejectsBroadMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-artifact")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer file.Close()

	if err := PreparePrivateFile(file, 0o640); err == nil {
		t.Fatal("PreparePrivateFile() accepted a group-readable mode")
	}
}
