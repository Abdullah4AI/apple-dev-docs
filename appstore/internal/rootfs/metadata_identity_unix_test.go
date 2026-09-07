//go:build darwin || linux

package rootfs

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/sys/unix"
)

func TestCaptureFileIdentityMetadataDetectsXattrDrift(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.txt")
	if err := os.WriteFile(path, []byte("metadata"), 0o640); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer file.Close()
	name := identityMetadataTestXattrName()
	if err := unix.Fsetxattr(int(file.Fd()), name, []byte("before"), 0); err != nil {
		if metadataOperationUnsupported(err) {
			t.Skipf("extended attributes are unavailable on this filesystem: %v", err)
		}
		t.Fatalf("set test extended attribute: %v", err)
	}

	before, err := captureFileIdentityMetadata(file)
	if err != nil {
		t.Fatalf("capture metadata before drift: %v", err)
	}
	if !before.xattrs.supported {
		t.Fatal("xattr setup succeeded but capture reported unsupported metadata")
	}
	unchanged, err := captureFileIdentityMetadata(file)
	if err != nil {
		t.Fatalf("capture unchanged metadata: %v", err)
	}
	if !sameFileIdentityMetadata(before, unchanged) {
		t.Fatal("unchanged metadata should compare equal")
	}

	if err := unix.Fsetxattr(int(file.Fd()), name, []byte("after"), 0); err != nil {
		t.Fatalf("change test extended attribute: %v", err)
	}
	after, err := captureFileIdentityMetadata(file)
	if err != nil {
		t.Fatalf("capture metadata after drift: %v", err)
	}
	if sameFileIdentityMetadata(before, after) {
		t.Fatal("xattr value drift should compare unequal")
	}
	if before.xattrs.equal(after.xattrs) {
		t.Fatal("xattr value drift should change the xattr digest")
	}
}

func TestStrictIdentityRejectsXattrDriftBeforePublication(t *testing.T) {
	requireStrictIdentityPlatform(t)
	dir := t.TempDir()
	root := mustRoot(t, dir)
	t.Cleanup(func() { _ = root.Close() })
	path := filepath.Join(dir, "metadata.txt")
	const original = "metadata"
	if err := os.WriteFile(path, []byte(original), 0o640); err != nil {
		t.Fatal(err)
	}

	identity, err := root.CaptureFile("metadata.txt")
	if err != nil {
		t.Fatalf("CaptureFile() error = %v", err)
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if err := unix.Fsetxattr(int(file.Fd()), identityMetadataTestXattrName(), []byte("drift"), 0); err != nil {
		_ = file.Close()
		if metadataOperationUnsupported(err) {
			t.Skipf("extended attributes are unavailable on this filesystem: %v", err)
		}
		t.Fatalf("set test extended attribute after capture: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	installed, err := root.ReplaceFileIfSame("metadata.txt", identity, []byte("replacement"), 0o640, true)
	if installed != nil {
		t.Fatal("ReplaceFileIfSame() returned an identity after xattr drift")
	}
	if !errors.Is(err, ErrFileIdentityChanged) {
		t.Fatalf("ReplaceFileIfSame() error = %v, want ErrFileIdentityChanged", err)
	}
	if got := mustRead(t, path); got != original {
		t.Fatalf("source after xattr drift = %q, want %q", got, original)
	}
}

func identityMetadataTestXattrName() string {
	if runtime.GOOS == "darwin" {
		return "com.rork.asc-rootfs-identity-test"
	}
	return "user.asc-rootfs-identity-test"
}

func TestCaptureFileIdentityMetadataRejectsMalformedXattrNameList(t *testing.T) {
	if _, err := parseIdentityXattrNames([]byte("user.test")); err == nil {
		t.Fatal("unterminated xattr name list should fail closed")
	}
}

func TestCaptureFileIdentityMetadataUnsupportedErrorClassification(t *testing.T) {
	if !metadataOperationUnsupported(unix.ENOTSUP) || !metadataOperationUnsupported(unix.EOPNOTSUPP) || !metadataOperationUnsupported(unix.ENOSYS) {
		t.Fatal("unsupported xattr errors should be classified as unsupported")
	}
	if metadataOperationUnsupported(unix.EPERM) || errors.Is(unix.EPERM, unix.ENOTSUP) {
		t.Fatal("permission errors must not be treated as unsupported metadata")
	}
}
