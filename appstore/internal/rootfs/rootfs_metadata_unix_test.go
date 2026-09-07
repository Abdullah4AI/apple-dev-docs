//go:build darwin || linux

package rootfs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/sys/unix"
)

func TestWriteFilePreservingModePreservesExtendedAttributes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	mustWrite(t, path, "original")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	name := "user.asc-rootfs-test"
	if runtime.GOOS == "darwin" {
		name = "com.rork.asc-rootfs-test"
	}
	value := []byte("metadata")
	if err := unix.Fsetxattr(int(file.Fd()), name, value, 0); err != nil {
		_ = file.Close()
		t.Skipf("extended attributes unavailable: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	root := mustRoot(t, dir)
	if err := root.WriteFilePreservingMode("existing.txt", []byte("replacement"), 0o644); err != nil {
		t.Fatalf("WriteFilePreservingMode() error = %v", err)
	}
	replaced, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(replaced) error = %v", err)
	}
	defer replaced.Close()
	size, err := unix.Fgetxattr(int(replaced.Fd()), name, nil)
	if err != nil {
		t.Fatalf("Fgetxattr(size) error = %v", err)
	}
	got := make([]byte, size)
	if _, err := unix.Fgetxattr(int(replaced.Fd()), name, got); err != nil {
		t.Fatalf("Fgetxattr() error = %v", err)
	}
	if string(got) != string(value) {
		t.Fatalf("extended attribute = %q, want %q", got, value)
	}
}

func TestWriteFilePreservingModePreservesSpecialBits(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS clears setuid/setgid during rooted replacement; Linux CI covers special-bit preservation")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "special.txt")
	mustWrite(t, path, "original")
	want := os.FileMode(0o755) | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
	if err := os.Chmod(path, want); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	initial, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(initial) error = %v", err)
	}
	mask := os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
	if got := initial.Mode() & mask; got != want {
		t.Skipf("filesystem does not preserve special mode bits (got %v, want %v)", got, want)
	}
	root := mustRoot(t, dir)
	if err := root.WriteFilePreservingMode("special.txt", []byte("replacement"), 0o600); err != nil {
		t.Fatalf("WriteFilePreservingMode() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode() & mask; got != want {
		t.Fatalf("mode = %v (%#o), want %v (%#o)", got, uint32(got.Perm()), want, uint32(want.Perm()))
	}
}
