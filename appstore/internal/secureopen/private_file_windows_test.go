//go:build windows

package secureopen

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestPrivateFileCreatorProtectsDACLAtCreation(t *testing.T) {
	directory := t.TempDir()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	const name = ".private-file-security-test"
	file, err := OpenNewPrivateFileNoFollowInRoot(root, name, 0o600)
	if err != nil {
		t.Fatalf("OpenNewPrivateFileNoFollowInRoot() error = %v", err)
	}
	defer file.Close()
	defer root.Remove(name)

	currentUser, err := privateFileCurrentUserSID()
	if err != nil {
		t.Fatalf("privateFileCurrentUserSID() error = %v", err)
	}
	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(file.Fd()),
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatalf("GetSecurityInfo(%q) error = %v", filepath.Join(directory, name), err)
	}
	if err := privateFileVerifyProtectedDACL(descriptor, currentUser); err != nil {
		t.Fatalf("privateFileVerifyProtectedDACL() error = %v, want owner-only DACL at creation", err)
	}
	if err := PreparePrivateFile(file, 0o600); err != nil {
		t.Fatalf("PreparePrivateFile() error = %v after creation-time protection", err)
	}
}

func TestPrivateFileOwnerAccessMaskUsesFileRights(t *testing.T) {
	mask := privateFileOwnerAccessMask()
	required := uint32(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE | windows.WRITE_DAC | windows.DELETE)
	if mask&required != required {
		t.Fatalf("privateFileOwnerAccessMask() = %#x, want read/write/delete rights %#x", mask, required)
	}
	if mask&windows.GENERIC_ALL != 0 {
		t.Fatalf("privateFileOwnerAccessMask() = %#x, want file-specific rights", mask)
	}
}
