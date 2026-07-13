//go:build darwin || linux

package telemetry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSpoolUsesSecurePermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".asc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create permissive spool directory: %v", err)
	}
	store := testSpoolStore(filepath.Join(dir, spoolFileName))
	if err := store.append(testSpoolRecord("event-01")); err != nil {
		t.Fatalf("append event: %v", err)
	}

	assertFileMode(t, dir, 0o700)
	assertFileMode(t, store.path, 0o600)
	assertFileMode(t, store.path+".lock", 0o600)
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %04o, want %04o", path, got, want)
	}
}
