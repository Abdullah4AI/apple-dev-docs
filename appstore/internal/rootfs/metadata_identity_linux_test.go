//go:build linux

package rootfs

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestCaptureFileIdentityMetadataDetectsLinuxACLDrift(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "acl.txt")
	if err := os.WriteFile(path, []byte("acl"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer file.Close()
	setLinuxIdentityACL(t, file, 4)

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

	setLinuxIdentityACL(t, file, 2)
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

func TestStrictIdentityRejectsLinuxACLDriftBeforePublication(t *testing.T) {
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
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer file.Close()
	setLinuxIdentityACL(t, file, 4)
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

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

func setLinuxIdentityACL(t *testing.T, file *os.File, namedUserPermissions uint16) {
	t.Helper()
	err := unix.Fsetxattr(int(file.Fd()), linuxPOSIXACLAccessIdentityAttribute, linuxIdentityACLBlob(namedUserPermissions), 0)
	if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.ENOSYS) {
		t.Skipf("POSIX ACLs are unavailable on this filesystem: %v", err)
	}
	if err != nil {
		t.Fatalf("set Linux POSIX access ACL: %v", err)
	}
	if size, err := unix.Fgetxattr(int(file.Fd()), linuxPOSIXACLAccessIdentityAttribute, nil); err != nil {
		t.Fatalf("verify Linux POSIX access ACL: %v", err)
	} else if size == 0 {
		t.Fatal("Linux POSIX access ACL disappeared after setting it")
	}
}

func linuxIdentityACLBlob(namedUserPermissions uint16) []byte {
	const (
		aclUserObj  = 0x01
		aclUser     = 0x02
		aclGroupObj = 0x04
		aclMask     = 0x10
		aclOther    = 0x20
		undefinedID = 0xFFFFFFFF
	)
	blob := make([]byte, 4+5*8)
	binary.LittleEndian.PutUint32(blob[0:4], 2)
	entries := []struct {
		tag  uint16
		perm uint16
		id   uint32
	}{
		{aclUserObj, 6, undefinedID},
		{aclUser, namedUserPermissions, 12345},
		{aclGroupObj, 0, undefinedID},
		{aclMask, 4, undefinedID},
		{aclOther, 0, undefinedID},
	}
	for index, entry := range entries {
		offset := 4 + index*8
		binary.LittleEndian.PutUint16(blob[offset:], entry.tag)
		binary.LittleEndian.PutUint16(blob[offset+2:], entry.perm)
		binary.LittleEndian.PutUint32(blob[offset+4:], entry.id)
	}
	return blob
}
