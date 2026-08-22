//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package assets

import (
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestEquivalentExistingPNGRejectsFIFO(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "screenshot.png")
	if err := unix.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("create FIFO: %v", err)
	}

	candidate := []byte("same screenshot bytes")
	fixtureFD, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		t.Fatalf("open FIFO fixture: %v", err)
	}
	if _, err := unix.Write(fixtureFD, candidate); err != nil {
		_ = unix.Close(fixtureFD)
		t.Fatalf("seed FIFO fixture: %v", err)
	}
	closed := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = unix.Close(fixtureFD)
		close(closed)
	}()

	result := make(chan bool, 1)
	go func() { result <- equivalentExistingPNG(path, candidate) }()
	var equivalent bool
	select {
	case equivalent = <-result:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out comparing FIFO fixture")
	}
	<-closed

	if equivalent {
		t.Fatal("equivalentExistingPNG() accepted a FIFO as an existing screenshot")
	}
}
