//go:build !windows
// +build !windows

package profiles

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestScanLocalProfiles_SkipsNonRegularFiles(t *testing.T) {
	installDir := t.TempDir()

	fifoPath := filepath.Join(installDir, "blocked.mobileprovision")
	if err := syscall.Mkfifo(fifoPath, 0o600); err != nil {
		t.Fatalf("Mkfifo(%q) error: %v", fifoPath, err)
	}

	type result struct {
		items   []localProfile
		skipped []localSkippedItem
		err     error
	}

	ch := make(chan result, 1)
	go func() {
		items, skipped, err := scanLocalProfiles(installDir, time.Now())
		ch <- result{items: items, skipped: skipped, err: err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("scanLocalProfiles() error: %v", res.err)
		}
		if len(res.items) != 0 {
			t.Fatalf("expected 0 items, got %d", len(res.items))
		}
		if len(res.skipped) != 1 {
			t.Fatalf("expected 1 skipped item, got %d", len(res.skipped))
		}
		if res.skipped[0].Path != fifoPath {
			t.Fatalf("skipped[0].Path=%q, want %q", res.skipped[0].Path, fifoPath)
		}
		if res.skipped[0].Reason != "refusing to read non-regular file" {
			t.Fatalf("skipped[0].Reason=%q, want %q", res.skipped[0].Reason, "refusing to read non-regular file")
		}
	case <-time.After(500 * time.Millisecond):
		// Regression safety: if scanLocalProfiles ever blocks on opening the FIFO,
		// unblock it so the test suite doesn't hang forever.
		// Opening the writer should complete immediately if the reader is blocked.
		if writer, err := os.OpenFile(fifoPath, os.O_WRONLY, 0); err == nil {
			_ = writer.Close()
		}
		t.Fatalf("scanLocalProfiles hung on a non-regular file (FIFO)")
	}
}
