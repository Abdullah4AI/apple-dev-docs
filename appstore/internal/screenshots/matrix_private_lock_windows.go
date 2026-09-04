//go:build windows

package screenshots

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

var matrixReOpenFile = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReOpenFile")

func lockMatrixPrivateAttemptDirectory(root *os.Root) error {
	return setMatrixPrivateAttemptDirectoryACL(root, "D:P(A;;GRGX;;;OW)")
}

func unlockMatrixPrivateAttemptDirectory(root *os.Root) error {
	return setMatrixPrivateAttemptDirectoryACL(root, "D:P(A;;GA;;;OW)")
}

func lockMatrixPrivateAttemptFile(path string) error {
	return setMatrixPrivateAttemptFileACL(path, "D:P(A;;GR;;;OW)")
}

func unlockMatrixPrivateAttemptFile(path string) error {
	return setMatrixPrivateAttemptFileACL(path, "D:P(A;;GA;;;OW)")
}

func lockMatrixPrivateAttemptFileHandle(file *os.File) error {
	return setMatrixPrivateAttemptFileACLHandle(file, "D:P(A;;GR;;;OW)")
}

func setMatrixPrivateAttemptDirectoryACL(root *os.Root, sddl string) error {
	if root == nil {
		return errors.New("private matrix attempt root is unavailable")
	}
	file, err := root.Open(".")
	if err != nil {
		return err
	}
	defer file.Close()
	// ReOpenFile cannot escalate a read-only os.Root handle to WRITE_DAC, and
	// directory reopen also requires FILE_FLAG_BACKUP_SEMANTICS. Open a fresh
	// handle against the already-pinned name instead of widening the original.
	handle, err := openMatrixDirectoryForDACL(file)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	if err := windows.SetSecurityInfo(
		handle,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	); err != nil {
		return fmt.Errorf("set private matrix attempt directory access control: %w", err)
	}
	return nil
}

func setMatrixPrivateAttemptFileACL(path, sddl string) error {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(
		name,
		windows.READ_CONTROL|windows.WRITE_DAC,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	return setMatrixPrivateAttemptFileACLHandleValue(handle, sddl)
}

func setMatrixPrivateAttemptFileACLHandle(file *os.File, sddl string) error {
	if file == nil {
		return errors.New("private matrix file is unavailable")
	}
	handle, err := reopenMatrixFileForDACL(file)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	return setMatrixPrivateAttemptFileACLHandleValue(handle, sddl)
}

func setMatrixPrivateAttemptFileACLHandleValue(handle windows.Handle, sddl string) error {
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	if err := windows.SetSecurityInfo(
		handle,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	); err != nil {
		return fmt.Errorf("set private matrix attempt file access control: %w", err)
	}
	return nil
}

func reopenMatrixFileForDACL(file *os.File) (windows.Handle, error) {
	handle, _, callErr := matrixReOpenFile.Call(
		file.Fd(),
		uintptr(windows.READ_CONTROL|windows.WRITE_DAC),
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

func reopenMatrixDirectoryForDACL(file *os.File) (windows.Handle, error) {
	return openMatrixDirectoryForDACL(file)
}

func openMatrixDirectoryForDACL(file *os.File) (windows.Handle, error) {
	if file == nil {
		return windows.InvalidHandle, errors.New("private matrix directory is unavailable")
	}
	name, err := windows.UTF16PtrFromString(file.Name())
	if err != nil {
		return windows.InvalidHandle, err
	}
	handle, err := windows.CreateFile(
		name,
		windows.READ_CONTROL|windows.WRITE_DAC,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return windows.InvalidHandle, err
	}
	return handle, nil
}
