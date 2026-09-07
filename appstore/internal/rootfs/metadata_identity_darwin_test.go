//go:build darwin

package rootfs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureFileIdentityMetadataDetectsDarwinACLDrift(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "acl.txt")
	if err := os.WriteFile(path, []byte("acl"), 0o640); err != nil {
		t.Fatal(err)
	}
	addDarwinIdentityACL(t, path, "everyone allow read")

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer file.Close()
	before, err := captureFileIdentityMetadata(file)
	if err != nil {
		t.Fatalf("capture ACL before drift: %v", err)
	}
	if !before.acl.supported {
		t.Fatal("ACL setup succeeded but capture reported unsupported metadata")
	}
	if before.acl.equal(supportedEmptyMetadataDigest()) {
		t.Fatal("ACL setup did not produce a non-empty ACL snapshot")
	}

	addDarwinIdentityACL(t, path, "everyone allow write")
	after, err := captureFileIdentityMetadata(file)
	if err != nil {
		t.Fatalf("capture ACL after drift: %v", err)
	}
	if sameFileIdentityMetadata(before, after) {
		t.Fatal("ACL drift should compare unequal")
	}
	if before.acl.equal(after.acl) {
		t.Fatal("ACL drift should change the ACL digest")
	}
}

func TestStrictIdentityRejectsDarwinACLDriftBeforePublication(t *testing.T) {
	requireStrictIdentityPlatform(t)
	dir := t.TempDir()
	root := mustRoot(t, dir)
	t.Cleanup(func() { _ = root.Close() })
	path := filepath.Join(dir, "acl.txt")
	const original = "acl"
	if err := os.WriteFile(path, []byte(original), 0o640); err != nil {
		t.Fatal(err)
	}

	identity, err := root.CaptureFile("acl.txt")
	if err != nil {
		t.Fatalf("CaptureFile() error = %v", err)
	}
	addDarwinIdentityACL(t, path, "everyone allow read")

	installed, err := root.ReplaceFileIfSame("acl.txt", identity, []byte("replacement"), 0o640, true)
	if installed != nil {
		t.Fatal("ReplaceFileIfSame() returned an identity after ACL drift")
	}
	if !errors.Is(err, ErrFileIdentityChanged) {
		t.Fatalf("ReplaceFileIfSame() error = %v, want ErrFileIdentityChanged", err)
	}
	if got := mustRead(t, path); got != original {
		t.Fatalf("source after ACL drift = %q, want %q", got, original)
	}
}

func addDarwinIdentityACL(t *testing.T, path, entry string) {
	t.Helper()
	output, err := exec.Command("chmod", "+a", entry, path).CombinedOutput()
	if err == nil {
		return
	}
	message := strings.ToLower(string(output))
	if strings.Contains(message, "operation not supported") || strings.Contains(message, "not supported") {
		t.Skipf("Darwin ACLs are unavailable on this filesystem: %v", err)
	}
	t.Fatalf("add Darwin ACL %q: %v\n%s", entry, err, output)
}
