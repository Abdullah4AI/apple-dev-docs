//go:build windows

package secureopen

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var reopenPrivateFileDACL = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReOpenFile")

// preparePrivateFile applies a protected owner-only DACL and verifies the
// resulting descriptor before the caller writes secret bytes. Windows does
// not use Go's Unix permission bits to restrict ordinary users, so Chmod alone
// would leave the inherited destination ACL in force.
func preparePrivateFile(file *os.File, perm os.FileMode) error {
	if file == nil {
		return fmt.Errorf("private file is nil")
	}
	if perm.Perm()&0o077 != 0 {
		return fmt.Errorf("private file mode %#o is not owner-only", perm.Perm())
	}
	currentUser, err := privateFileCurrentUserSID()
	if err != nil {
		return fmt.Errorf("protect private file permissions: %w", err)
	}

	var pinner runtime.Pinner
	pinner.Pin(currentUser)
	defer pinner.Unpin()
	acl, err := privateFileOwnerACL(currentUser)
	if err != nil {
		return fmt.Errorf("protect private file permissions: %w", err)
	}

	handle, err := reopenPrivateFileForDACL(file)
	if err != nil {
		return fmt.Errorf("protect private file permissions: %w", err)
	}
	defer windows.CloseHandle(handle)
	if err := windows.SetSecurityInfo(
		handle,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	); err != nil {
		return fmt.Errorf("protect private file permissions: %w", err)
	}

	descriptor, err := windows.GetSecurityInfo(
		handle,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return fmt.Errorf("verify private file permissions: %w", err)
	}
	if err := privateFileVerifyProtectedDACL(descriptor, currentUser); err != nil {
		return fmt.Errorf("verify private file permissions: %w", err)
	}
	runtime.KeepAlive(acl)
	return nil
}

// privateFileOwnerAccessMask uses file-specific rights rather than generic
// access bits. The verifier compares the exact rights that the file needs.
func privateFileOwnerAccessMask() uint32 {
	return uint32(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE | windows.WRITE_DAC | windows.DELETE)
}

func privateFileOwnerACL(currentUser *windows.SID) (*windows.ACL, error) {
	if currentUser == nil || !currentUser.IsValid() {
		return nil, fmt.Errorf("current user SID is unavailable")
	}
	return windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.ACCESS_MASK(privateFileOwnerAccessMask()),
		AccessMode:        windows.SET_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(currentUser),
		},
	}}, nil)
}

// openNewPrivateFileNoFollowInRoot attaches the owner-only DACL to the native
// create operation. The generated name and rooted parent are supplied by
// secureopen, so this callback cannot select a different path or relax the
// exclusive no-follow contract.
func openNewPrivateFileNoFollowInRoot(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	if root == nil {
		return nil, fmt.Errorf("create private file: missing rooted parent")
	}
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || filepath.Clean(name) != name {
		return nil, fmt.Errorf("create private file: invalid generated name %q", name)
	}
	if perm.Perm()&0o077 != 0 {
		return nil, fmt.Errorf("create private file mode %#o is not owner-only", perm.Perm())
	}

	currentUser, err := privateFileCurrentUserSID()
	if err != nil {
		return nil, fmt.Errorf("create private file: %w", err)
	}
	var pinner runtime.Pinner
	pinner.Pin(currentUser)
	defer pinner.Unpin()

	acl, err := privateFileOwnerACL(currentUser)
	if err != nil {
		return nil, fmt.Errorf("create private file: %w", err)
	}
	securityDescriptor, err := windows.NewSecurityDescriptor()
	if err != nil {
		return nil, fmt.Errorf("create private file security descriptor: %w", err)
	}
	if err := securityDescriptor.SetOwner(currentUser, false); err != nil {
		return nil, fmt.Errorf("create private file owner: %w", err)
	}
	if err := securityDescriptor.SetDACL(acl, true, false); err != nil {
		return nil, fmt.Errorf("create private file DACL: %w", err)
	}
	if err := securityDescriptor.SetControl(windows.SE_DACL_PROTECTED, windows.SE_DACL_PROTECTED); err != nil {
		return nil, fmt.Errorf("protect private file DACL: %w", err)
	}

	parent, err := root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("open private file parent: %w", err)
	}
	defer parent.Close()
	raw, err := parent.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("open private file parent handle: %w", err)
	}
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return nil, fmt.Errorf("create private file name: %w", err)
	}

	var created windows.Handle
	var createErr error
	if err := raw.Control(func(parentHandle uintptr) {
		createErr = createPrivateFile(
			windows.Handle(parentHandle),
			objectName,
			securityDescriptor,
			&created,
		)
	}); err != nil {
		return nil, fmt.Errorf("open private file parent handle: %w", err)
	}
	if createErr != nil {
		return nil, fmt.Errorf("create private file: %w", createErr)
	}
	if created == 0 || created == windows.InvalidHandle {
		return nil, fmt.Errorf("create private file: native create returned an invalid handle")
	}
	file := os.NewFile(uintptr(created), filepath.Join(root.Name(), name))
	if file == nil {
		_ = windows.CloseHandle(created)
		return nil, fmt.Errorf("create private file: cannot wrap native handle")
	}
	runtime.KeepAlive(objectName)
	runtime.KeepAlive(securityDescriptor)
	runtime.KeepAlive(acl)
	return file, nil
}

func createPrivateFile(parent windows.Handle, objectName *windows.NTUnicodeString, securityDescriptor *windows.SECURITY_DESCRIPTOR, created *windows.Handle) error {
	objectAttributes := windows.OBJECT_ATTRIBUTES{
		Length:             uint32(unsafe.Sizeof(windows.OBJECT_ATTRIBUTES{})),
		RootDirectory:      parent,
		ObjectName:         objectName,
		Attributes:         windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE,
		SecurityDescriptor: securityDescriptor,
	}
	var status windows.IO_STATUS_BLOCK
	return windows.NtCreateFile(
		created,
		privateFileOwnerAccessMask(),
		&objectAttributes,
		&status,
		nil,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		windows.FILE_CREATE,
		windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT,
		0,
		0,
	)
}

func privateFileCurrentUserSID() (*windows.SID, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	if user == nil || user.User.Sid == nil || !user.User.Sid.IsValid() {
		return nil, fmt.Errorf("current user SID is unavailable")
	}
	return user.User.Sid, nil
}

func privateFileVerifyProtectedDACL(descriptor *windows.SECURITY_DESCRIPTOR, currentUser *windows.SID) error {
	if descriptor == nil || currentUser == nil {
		return fmt.Errorf("missing security descriptor")
	}
	owner, _, err := descriptor.Owner()
	if err != nil {
		return err
	}
	if owner == nil || !owner.Equals(currentUser) {
		return fmt.Errorf("file is not owned by the current user")
	}
	control, _, err := descriptor.Control()
	if err != nil {
		return err
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return fmt.Errorf("DACL is inherited")
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	if dacl == nil || dacl.AceCount == 0 {
		return fmt.Errorf("DACL is missing")
	}
	if dacl.AceCount != 1 {
		return fmt.Errorf("DACL contains entries for another account")
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &ace); err != nil {
		return err
	}
	if ace == nil || ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags != 0 {
		return fmt.Errorf("DACL contains an unsupported access entry")
	}
	aceSID := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if aceSID == nil || !aceSID.Equals(currentUser) {
		return fmt.Errorf("DACL grants access to another account")
	}
	required := uint32(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE)
	if uint32(ace.Mask)&required != required {
		return fmt.Errorf("current user cannot read and write the file")
	}
	return nil
}

func reopenPrivateFileForDACL(file *os.File) (windows.Handle, error) {
	handle, _, callErr := reopenPrivateFileDACL.Call(
		file.Fd(),
		uintptr(windows.WRITE_DAC|windows.READ_CONTROL),
		uintptr(windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE),
		0,
	)
	if reopened := windows.Handle(handle); reopened != windows.InvalidHandle {
		return reopened, nil
	}
	if callErr != nil && callErr != syscall.Errno(0) {
		return windows.InvalidHandle, callErr
	}
	return windows.InvalidHandle, syscall.EINVAL
}
