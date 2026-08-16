//go:build windows

package distribute

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func openDistributionRunLockFile(runRoot *os.Root, name string) (*os.File, error) {
	file, err := runRoot.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err == nil {
		return file, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	return runRoot.OpenFile(name, os.O_RDWR, 0)
}

func tryDistributionRunFileLock(file *os.File) (bool, error) {
	var overlapped windows.Overlapped
	err := windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		&overlapped,
	)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
		return false, nil
	}
	return false, err
}

func unlockDistributionRunFile(file *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlapped)
}

func validateDistributionRunLockPlatform(file *os.File) error {
	handle := windows.Handle(file.Fd())
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("protected input is a reparse point")
	}
	if info.NumberOfLinks != 1 {
		return fmt.Errorf("protected input must not be a hard link")
	}

	securityDescriptor, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("inspect protected input owner: %w", err)
	}
	if securityDescriptor == nil {
		return fmt.Errorf("inspect protected input owner: missing security descriptor")
	}
	owner, _, err := securityDescriptor.Owner()
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
