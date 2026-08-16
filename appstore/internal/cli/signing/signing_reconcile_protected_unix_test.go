//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package signing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadProtectedFileRejectsForeignOwnerWhenTestCanChown(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("changing file ownership requires root")
	}
	path := filepath.Join(t.TempDir(), "devices.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	foreignUID := 1
	if foreignUID == os.Geteuid() {
		foreignUID = 2
	}
	if err := os.Chown(path, foreignUID, -1); err != nil {
		t.Skipf("cannot create foreign-owned fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.Chown(path, os.Geteuid(), -1) })
	if _, err := readProtectedFile(path); err == nil || !strings.Contains(err.Error(), "owned by the current user") {
		t.Fatalf("foreign-owned protected input error=%v", err)
	}
}
