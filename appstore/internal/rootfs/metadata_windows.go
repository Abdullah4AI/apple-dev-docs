//go:build windows

package rootfs

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"

	"golang.org/x/sys/windows"
)

var reOpenFile = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReOpenFile")

func copyReplacementMetadata(destination, source *os.File, info os.FileInfo) error {
	if err := destination.Chmod(info.Mode().Perm()); err != nil {
		return fmt.Errorf("preserve replacement permissions: %w", err)
	}

	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(source.Fd()),
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		// Filesystems without persistent ACL support have no DACL metadata to
		// preserve; keep the pre-existing mode-only behavior there.
		if errors.Is(err, windows.ERROR_INVALID_FUNCTION) || errors.Is(err, windows.ERROR_NOT_SUPPORTED) {
			return nil
		}
		return fmt.Errorf("read replacement access control list: %w", err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return fmt.Errorf("read replacement access control list: %w", err)
	}
	control, _, err := descriptor.Control()
	if err != nil {
		return fmt.Errorf("inspect replacement access control list: %w", err)
	}
	shouldSet, err := shouldSetReplacementDACL(control, dacl)
	if err != nil {
		return fmt.Errorf("inspect replacement access control list: %w", err)
	}
	if !shouldSet {
		// The replacement already inherits the parent DACL. Copying that
		// inherited-only ACL with SetSecurityInfo fails with ACCESS_DENIED.
		return nil
	}

	destinationHandle, err := reopenForDACL(destination)
	if err != nil {
		return fmt.Errorf("open replacement for access control update: %w", err)
	}
	defer windows.CloseHandle(destinationHandle)

	err = windows.SetSecurityInfo(
		destinationHandle,
		windows.SE_FILE_OBJECT,
		daclSecurityInformation(control),
		nil,
		nil,
		dacl,
		nil,
	)
	runtime.KeepAlive(descriptor)
	if err != nil {
		return fmt.Errorf("preserve replacement access control list: %w", err)
	}
	return nil
}

func restoreReplacementMode(destination *os.File, info os.FileInfo) error {
	return destination.Chmod(info.Mode().Perm())
}

func shouldSetReplacementDACL(control windows.SECURITY_DESCRIPTOR_CONTROL, dacl *windows.ACL) (bool, error) {
	if dacl == nil {
		// SECURITY_DESCRIPTOR.DACL() returns (nil, _, nil) for a present NULL
		// DACL, which is fully permissive and must be preserved. A missing
		// DACL is reported as an error by DACL() before we get here.
		return true, nil
	}
	if dacl.AceCount == 0 {
		// An empty DACL denies all access. Skipping the copy would leave the
		// replacement with its inherited parent ACL and broaden access.
		return true, nil
	}
	if control&windows.SE_DACL_PROTECTED != 0 {
		return true, nil
	}
	return hasExplicitACE(dacl)
}

func hasExplicitACE(dacl *windows.ACL) (bool, error) {
	if dacl == nil {
		return false, nil
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return false, err
		}
		if ace.Header.AceFlags&windows.INHERITED_ACE == 0 {
			return true, nil
		}
	}
	return false, nil
}

func reopenForDACL(file *os.File) (windows.Handle, error) {
	handle, _, callErr := reOpenFile.Call(
		file.Fd(),
		uintptr(windows.WRITE_DAC),
		uintptr(windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE),
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

func daclSecurityInformation(control windows.SECURITY_DESCRIPTOR_CONTROL) windows.SECURITY_INFORMATION {
	information := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION)
	if control&windows.SE_DACL_PROTECTED != 0 {
		return information | windows.PROTECTED_DACL_SECURITY_INFORMATION
	}
	return information | windows.UNPROTECTED_DACL_SECURITY_INFORMATION
}
