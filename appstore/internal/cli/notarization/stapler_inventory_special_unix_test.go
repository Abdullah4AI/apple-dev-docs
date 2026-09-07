//go:build !windows

package notarization

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func makeStaplerSpecialEntry(t *testing.T, directory string) {
	t.Helper()
	if err := unix.Mkfifo(filepath.Join(directory, "named-pipe"), 0o600); err != nil {
		t.Skipf("named-pipe creation unavailable: %v", err)
	}
}
