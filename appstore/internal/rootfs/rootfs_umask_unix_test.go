//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package rootfs

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestWriteFilePreservingModeHonorsUmaskForNewFile(t *testing.T) {
	oldUmask := syscall.Umask(0o077)
	defer syscall.Umask(oldUmask)

	dir := t.TempDir()
	root := mustRoot(t, dir)
	if err := root.WriteFilePreservingMode("fresh.txt", []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFilePreservingMode() error = %v", err)
	}
	info, err := os.Lstat(filepath.Join(dir, "fresh.txt"))
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("new mode = %v, want umask-restricted 0600", info.Mode().Perm())
	}
}

func TestWriteFileRefusesToReplaceSpecialFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("Mkfifo() error = %v", err)
	}

	root := mustRoot(t, dir)
	if err := root.WriteFile("pipe", []byte("replacement"), 0o600); err == nil {
		t.Fatal("WriteFile() error = nil, want special-file refusal")
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("destination mode = %v, want named pipe preserved", info.Mode())
	}
}

func TestAppendFileHonorsUmaskForNewFile(t *testing.T) {
	oldUmask := syscall.Umask(0o077)
	defer syscall.Umask(oldUmask)

	dir := t.TempDir()
	root := mustRoot(t, dir)
	if err := root.AppendFile("fresh.log", []byte("entry\n"), 0o644); err != nil {
		t.Fatalf("AppendFile() error = %v", err)
	}
	info, err := os.Lstat(filepath.Join(dir, "fresh.log"))
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("new mode = %v, want umask-restricted 0600", info.Mode().Perm())
	}
}
