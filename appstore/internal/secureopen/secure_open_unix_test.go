//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package secureopen

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestOpenExistingNoFollow_FIFO_DoesNotBlock(t *testing.T) {
	dir := t.TempDir()
	fifoPath := filepath.Join(dir, "test.fifo")

	if err := unix.Mkfifo(fifoPath, 0o600); err != nil {
		t.Skipf("mkfifo not supported: %v", err)
	}

	done := make(chan struct{})
	var (
		f   *os.File
		err error
	)

	go func() {
		f, err = OpenExistingNoFollow(fifoPath)
		close(done)
	}()

	select {
	case <-done:
		if err != nil {
			t.Fatalf("OpenExistingNoFollow() error: %v", err)
		}
		_ = f.Close()
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("OpenExistingNoFollow() blocked opening FIFO")
	}
}
