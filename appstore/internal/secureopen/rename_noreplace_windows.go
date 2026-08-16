//go:build windows

package secureopen

import (
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type fileRenameInformation struct {
	ReplaceIfExists uint32
	RootDirectory   windows.Handle
	FileNameLength  uint32
	FileName        [1]uint16
}

// RenameNoReplaceInRoot atomically renames oldName to newName beneath root and
// fails when newName already exists.
func RenameNoReplaceInRoot(root *os.Root, oldName, newName string) error {
	const op = "NtSetInformationFile"
	if err := validateRenameNoReplaceNames(oldName, newName); err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}

	directory, err := root.Open(".")
	if err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}
	defer directory.Close()
	raw, err := directory.SyscallConn()
	if err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}
	var renameErr error
	if err := raw.Control(func(handle uintptr) {
		renameErr = renameNoReplaceWindows(windows.Handle(handle), oldName, newName)
	}); err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}
	renameErr = windowsError(renameErr)
	if errors.Is(renameErr, windows.ERROR_INVALID_FUNCTION) || errors.Is(renameErr, windows.ERROR_NOT_SUPPORTED) || errors.Is(renameErr, windows.ERROR_CALL_NOT_IMPLEMENTED) {
		return unsupportedRenameNoReplaceError(op, oldName, newName, renameErr)
	}
	return renameNoReplaceError(op, oldName, newName, renameErr)
}

func renameNoReplaceWindows(parent windows.Handle, oldName, newName string) error {
	objectName, err := windows.NewNTUnicodeString(oldName)
	if err != nil {
		return err
	}
	objectAttributes := windows.OBJECT_ATTRIBUTES{
		Length:        uint32(unsafe.Sizeof(windows.OBJECT_ATTRIBUTES{})),
		RootDirectory: parent,
		ObjectName:    objectName,
		Attributes:    windows.OBJ_CASE_INSENSITIVE,
	}
	var source windows.Handle
	var status windows.IO_STATUS_BLOCK
	createOptions := uint32(windows.FILE_OPEN_REPARSE_POINT | windows.FILE_OPEN_FOR_BACKUP_INTENT | windows.FILE_SYNCHRONOUS_IO_NONALERT)
	// FILE_NON_DIRECTORY_FILE made this primitive unusable for atomic bundle
	// directory publication. Omitting both type constraints lets the no-follow
	// handle open either a regular file or a directory while preserving the
	// existing rename contract.
	err = windows.NtCreateFile(
		&source,
		windows.DELETE|windows.SYNCHRONIZE,
		&objectAttributes,
		&status,
		nil,
		0,
		windows.FILE_SHARE_DELETE|windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		windows.FILE_OPEN,
		createOptions,
		0,
		0,
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(source)

	buffer, err := buildFileRenameInformation(newName)
	if err != nil {
		return err
	}

	return windows.NtSetInformationFile(
		source,
		&status,
		&buffer[0],
		uint32(len(buffer)),
		windows.FileRenameInformation,
	)
}

func buildFileRenameInformation(newName string) ([]byte, error) {
	newNameUTF16, err := windows.UTF16FromString(newName)
	if err != nil {
		return nil, err
	}
	fileNameLength := (len(newNameUTF16) - 1) * 2
	var layout fileRenameInformation
	buffer := make([]byte, int(unsafe.Offsetof(layout.FileName))+fileNameLength)
	info := (*fileRenameInformation)(unsafe.Pointer(&buffer[0]))
	// Keep RootDirectory null. Windows defines a simple target name with a null
	// root as a rename within the source handle's directory, and SMB requires
	// this field to be zero for network operations.
	info.FileNameLength = uint32(fileNameLength)
	copy(unsafe.Slice(&info.FileName[0], fileNameLength/2), newNameUTF16[:len(newNameUTF16)-1])
	return buffer, nil
}

func windowsError(err error) error {
	if status, ok := err.(windows.NTStatus); ok {
		return status.Errno()
	}
	return err
}
