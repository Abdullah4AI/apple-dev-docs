//go:build darwin || linux

package rootfs

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// alternateGroup returns a supplementary group of the current process that is
// not the group already owning info, so a test can change ownership without
// privileges.
func alternateGroup(t *testing.T, info os.FileInfo) int {
	t.Helper()
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skipf("stat type %T does not expose ownership", info.Sys())
	}
	groups, err := os.Getgroups()
	if err != nil {
		t.Fatalf("Getgroups() error = %v", err)
	}
	for _, group := range groups {
		if group >= 0 && uint32(group) != stat.Gid {
			return group
		}
	}
	t.Skip("no alternate supplementary group is available for an ownership change")
	return 0
}

func TestReplaceFileIfSameRejectsOwnershipDrift(t *testing.T) {
	requireStrictIdentityPlatform(t)
	dir := t.TempDir()
	root := mustRoot(t, dir)
	t.Cleanup(func() { _ = root.Close() })
	path := filepath.Join(dir, "settings.xcconfig")
	const original = "CODE_SIGN_STYLE = Automatic\n"
	if err := os.WriteFile(path, []byte(original), 0o640); err != nil {
		t.Fatal(err)
	}

	identity, err := root.CaptureFile("settings.xcconfig")
	if err != nil {
		t.Fatalf("CaptureFile() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(source) error = %v", err)
	}
	// A group change alters neither the permission bits nor the modification
	// time, so it is invisible to every other strict comparison.
	group := alternateGroup(t, info)
	if err := os.Chown(path, -1, group); err != nil {
		t.Skipf("change file group: %v", err)
	}

	installed, err := root.ReplaceFileIfSame("settings.xcconfig", identity, []byte("CODE_SIGN_STYLE = Manual\n"), 0o640, true)
	if !errors.Is(err, ErrFileIdentityChanged) {
		t.Fatalf("ReplaceFileIfSame() error = %v, want ErrFileIdentityChanged", err)
	}
	if installed != nil {
		t.Fatal("ReplaceFileIfSame() returned an identity after ownership drift")
	}
	if got := mustRead(t, path); got != original {
		t.Fatalf("source after ownership drift = %q, want %q", got, original)
	}
	matches, globErr := filepath.Glob(filepath.Join(dir, ".asc-tmp-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("staging or quarantine entries remain after ownership drift: %v", matches)
	}
}

func TestCheckFileIdentityRejectsOwnershipDrift(t *testing.T) {
	requireStrictIdentityPlatform(t)
	dir := t.TempDir()
	root := mustRoot(t, dir)
	t.Cleanup(func() { _ = root.Close() })
	path := filepath.Join(dir, "receipt.json")
	if err := os.WriteFile(path, []byte(`{"completed":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	identity, err := root.CaptureFile("receipt.json")
	if err != nil {
		t.Fatalf("CaptureFile() error = %v", err)
	}
	if err := root.CheckFileIdentity("receipt.json", identity); err != nil {
		t.Fatalf("CheckFileIdentity() error = %v before ownership drift", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(receipt) error = %v", err)
	}
	group := alternateGroup(t, info)
	if err := os.Chown(path, -1, group); err != nil {
		t.Skipf("change file group: %v", err)
	}

	if err := root.CheckFileIdentity("receipt.json", identity); !errors.Is(err, ErrFileIdentityChanged) {
		t.Fatalf("CheckFileIdentity() error = %v, want ErrFileIdentityChanged", err)
	}
}
