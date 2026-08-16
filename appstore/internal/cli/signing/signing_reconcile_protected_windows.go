//go:build windows

package signing

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func validateReconcileProtectedFilePlatform(file *os.File, _ os.FileInfo) error {
	handle := windows.Handle(file.Fd())
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return fmt.Errorf("inspect protected input link count: %w", err)
	}
	if info.NumberOfLinks != 1 {
		return fmt.Errorf("protected input must not have multiple hard links")
	}
	descriptor, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("inspect protected input owner: %w", err)
	}
	if descriptor == nil {
		return fmt.Errorf("inspect protected input owner: missing security descriptor")
	}
	owner, _, err := descriptor.Owner()
	if err != nil {
		return fmt.Errorf("inspect protected input owner: %w", err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return fmt.Errorf("inspect current process owner: %w", err)
	}
	if owner == nil || user == nil || user.User.Sid == nil || !windows.EqualSid(owner, user.User.Sid) {
		return fmt.Errorf("protected input must be owned by the current user")
	}
	return nil
}
