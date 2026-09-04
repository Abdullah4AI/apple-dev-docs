//go:build windows

package certificates

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var reopenCertificateExportDACL = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReOpenFile")

func validateCertificateExportProtectedFile(file *os.File, _ os.FileInfo, label string) error {
	if file == nil {
		return permissionErrorForCertificateExport(label)
	}
	currentUser, err := certificateExportCurrentUserSID()
	if err != nil {
		return fmt.Errorf("%s permissions cannot be verified: %w", label, err)
	}
	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(file.Fd()),
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return fmt.Errorf("%s permissions cannot be verified: %w", label, err)
	}
	if err := certificateExportVerifyProtectedDACL(descriptor, currentUser, true); err != nil {
		return fmt.Errorf("%s permissions are not restricted to the current user: %w", label, err)
	}
	return nil
}

func prepareCertificateExportOutput(file *os.File) error {
	if file == nil {
		return fmt.Errorf("output permissions cannot be protected")
	}
	currentUser, err := certificateExportCurrentUserSID()
	if err != nil {
		return fmt.Errorf("protect output permissions: %w", err)
	}

	var pinner runtime.Pinner
	pinner.Pin(currentUser)
	defer pinner.Unpin()
	acl, err := certificateExportOwnerACL(currentUser)
	if err != nil {
		return fmt.Errorf("protect output permissions: %w", err)
	}

	handle, err := reopenCertificateExportFileForDACL(file)
	if err != nil {
		return fmt.Errorf("protect output permissions: %w", err)
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
		return fmt.Errorf("protect output permissions: %w", err)
	}

	descriptor, err := windows.GetSecurityInfo(
		handle,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return fmt.Errorf("verify output permissions: %w", err)
	}
	if err := certificateExportVerifyProtectedDACL(descriptor, currentUser, false); err != nil {
		return fmt.Errorf("verify output permissions: %w", err)
	}
	runtime.KeepAlive(acl)
	return nil
}

// certificateExportOwnerAccessMask uses file-specific rights rather than a
// generic access bit. SetEntriesInAcl preserves the mask supplied by callers;
// generic bits are not expanded before the verifier inspects file rights.
func certificateExportOwnerAccessMask() uint32 {
	return uint32(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE | windows.WRITE_DAC | windows.DELETE)
}

func certificateExportOwnerACL(currentUser *windows.SID) (*windows.ACL, error) {
	if currentUser == nil || !currentUser.IsValid() {
		return nil, fmt.Errorf("current user SID is unavailable")
	}
	return windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.ACCESS_MASK(certificateExportOwnerAccessMask()),
		AccessMode:        windows.SET_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(currentUser),
		},
	}}, nil)
}

// createCertificateExportStagingFile creates the staging file with its
// owner-only DACL attached to the native create operation. The generated name
// and rooted parent are supplied by secureopen; this callback cannot select a
// different path or relax the no-follow/exclusive creation contract.
func createCertificateExportStagingFile(root *os.Root, name string, _ os.FileMode) (*os.File, error) {
	if root == nil {
		return nil, fmt.Errorf("create protected output: missing rooted parent")
	}
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || filepath.Clean(name) != name {
		return nil, fmt.Errorf("create protected output: invalid generated name %q", name)
	}

	currentUser, err := certificateExportCurrentUserSID()
	if err != nil {
		return nil, fmt.Errorf("create protected output: %w", err)
	}
	var pinner runtime.Pinner
	pinner.Pin(currentUser)
	defer pinner.Unpin()

	acl, err := certificateExportOwnerACL(currentUser)
	if err != nil {
		return nil, fmt.Errorf("create protected output: %w", err)
	}
	securityDescriptor, err := windows.NewSecurityDescriptor()
	if err != nil {
		return nil, fmt.Errorf("create protected output security descriptor: %w", err)
	}
	if err := securityDescriptor.SetOwner(currentUser, false); err != nil {
		return nil, fmt.Errorf("create protected output owner: %w", err)
	}
	if err := securityDescriptor.SetDACL(acl, true, false); err != nil {
		return nil, fmt.Errorf("create protected output DACL: %w", err)
	}
	if err := securityDescriptor.SetControl(windows.SE_DACL_PROTECTED, windows.SE_DACL_PROTECTED); err != nil {
		return nil, fmt.Errorf("protect output DACL: %w", err)
	}

	parent, err := root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("open protected output parent: %w", err)
	}
	defer parent.Close()
	raw, err := parent.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("open protected output parent handle: %w", err)
	}
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return nil, fmt.Errorf("create protected output name: %w", err)
	}

	var created windows.Handle
	var createErr error
	if err := raw.Control(func(parentHandle uintptr) {
		createErr = createCertificateExportFile(
			windows.Handle(parentHandle),
			objectName,
			securityDescriptor,
			&created,
		)
	}); err != nil {
		return nil, fmt.Errorf("open protected output parent handle: %w", err)
	}
	if createErr != nil {
		return nil, fmt.Errorf("create protected output: %w", createErr)
	}
	if created == 0 || created == windows.InvalidHandle {
		return nil, fmt.Errorf("create protected output: native create returned an invalid handle")
	}
	file := os.NewFile(uintptr(created), filepath.Join(root.Name(), name))
	if file == nil {
		_ = windows.CloseHandle(created)
		return nil, fmt.Errorf("create protected output: cannot wrap native handle")
	}
	runtime.KeepAlive(objectName)
	runtime.KeepAlive(securityDescriptor)
	runtime.KeepAlive(acl)
	return file, nil
}

func createCertificateExportFile(parent windows.Handle, objectName *windows.NTUnicodeString, securityDescriptor *windows.SECURITY_DESCRIPTOR, created *windows.Handle) error {
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
		certificateExportOwnerAccessMask(),
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

func certificateExportCurrentUserSID() (*windows.SID, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	if user == nil || user.User.Sid == nil || !user.User.Sid.IsValid() {
		return nil, fmt.Errorf("current user SID is unavailable")
	}
	return user.User.Sid, nil
}

// certificateExportVerifyProtectedDACL verifies that a security descriptor
// restricts a file to the current user. In input mode
// (allowTrustedSystemEntries), inherited DACLs are acceptable as long as every
// effective entry belongs to the current user or a trusted system account, so
// keys written by `asc certificates csr generate` and normally created
// password files validate. Output mode remains strict: the DACL this command
// applies must be protected, non-inherited, and owner-only.
func certificateExportVerifyProtectedDACL(descriptor *windows.SECURITY_DESCRIPTOR, currentUser *windows.SID, allowTrustedSystemEntries bool) error {
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
	if !allowTrustedSystemEntries {
		control, _, controlErr := descriptor.Control()
		if controlErr != nil {
			return controlErr
		}
		if control&windows.SE_DACL_PROTECTED == 0 {
			return fmt.Errorf("DACL is inherited")
		}
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	if dacl == nil || dacl.AceCount == 0 {
		return fmt.Errorf("DACL is missing")
	}

	var permittedACEFlags uint8
	if allowTrustedSystemEntries {
		// Inherited allow entries are the only additional shape accepted for
		// inputs; inheritance-propagation and audit flags stay rejected.
		permittedACEFlags = uint8(windows.INHERITED_ACE)
	}
	trustedSystem, trustedAdministrators := certificateExportTrustedSIDs()
	currentUserEntries := 0
	var currentUserAccess uint32
	for index := uint16(0); index < dacl.AceCount; index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, uint32(index), &ace); err != nil {
			return err
		}
		if ace == nil || ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags&^permittedACEFlags != 0 {
			return fmt.Errorf("DACL contains an unsupported access entry")
		}
		aceSID := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if aceSID == nil {
			return fmt.Errorf("DACL contains an invalid access entry")
		}
		if aceSID.Equals(currentUser) {
			currentUserEntries++
			currentUserAccess |= uint32(ace.Mask)
			continue
		}
		if allowTrustedSystemEntries && ((trustedSystem != nil && aceSID.Equals(trustedSystem)) || (trustedAdministrators != nil && aceSID.Equals(trustedAdministrators))) {
			continue
		}
		return fmt.Errorf("DACL grants access to another account")
	}
	if currentUserEntries == 0 {
		return fmt.Errorf("DACL does not grant access to the current user")
	}
	if !allowTrustedSystemEntries && currentUserEntries != 1 {
		return fmt.Errorf("DACL contains duplicate current-user entries")
	}
	if allowTrustedSystemEntries {
		// Inputs are only read, so a hardened read-only key or password file
		// (the Windows analog of Unix mode 0400) stays usable.
		if currentUserAccess&uint32(windows.FILE_GENERIC_READ) != uint32(windows.FILE_GENERIC_READ) {
			return fmt.Errorf("current user cannot read the file")
		}
		return nil
	}
	required := uint32(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE)
	if currentUserAccess&required != required {
		return fmt.Errorf("current user cannot read and write the file")
	}
	return nil
}

func certificateExportTrustedSIDs() (*windows.SID, *windows.SID) {
	system, systemErr := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	administrators, administratorsErr := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if systemErr != nil {
		system = nil
	}
	if administratorsErr != nil {
		administrators = nil
	}
	return system, administrators
}

func reopenCertificateExportFileForDACL(file *os.File) (windows.Handle, error) {
	handle, _, callErr := reopenCertificateExportDACL.Call(
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

func permissionErrorForCertificateExport(label string) error {
	return fmt.Errorf("%s permissions must be 0600 or more restrictive", label)
}
