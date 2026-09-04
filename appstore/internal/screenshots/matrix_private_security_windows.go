//go:build windows

package screenshots

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

const matrixOwnerOnlyRandomNameAttempts = 64

// matrixOwnerOnlySecurityAttributes returns a protected owner-only DACL for
// objects that must never be visible through a broad inherited Windows ACL.
// The descriptor is attached to CreateDirectory/CreateFile so the object is
// protected from its first externally visible instant.
func matrixOwnerOnlySecurityAttributes() (*windows.SecurityAttributes, error) {
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;;GA;;;OW)")
	if err != nil {
		return nil, fmt.Errorf("create owner-only security descriptor: %w", err)
	}
	return &windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
	}, nil
}

func createMatrixOwnerOnlyTempDir(prefix string) (string, error) {
	security, err := matrixOwnerOnlySecurityAttributes()
	if err != nil {
		return "", err
	}
	for attempt := 0; attempt < matrixOwnerOnlyRandomNameAttempts; attempt++ {
		var suffix [16]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return "", fmt.Errorf("generate owner-only temporary name: %w", err)
		}
		path := filepath.Join(os.TempDir(), prefix+hex.EncodeToString(suffix[:]))
		name, err := windows.UTF16PtrFromString(path)
		if err != nil {
			return "", fmt.Errorf("encode owner-only temporary path: %w", err)
		}
		if err := windows.CreateDirectory(name, security); err == nil {
			return path, nil
		} else if errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_FILE_EXISTS) {
			continue
		} else {
			return "", fmt.Errorf("create owner-only temporary directory: %w", err)
		}
	}
	return "", errors.New("create owner-only temporary directory: random-name collision limit exceeded")
}

func createMatrixOwnerOnlyDirectory(path string) error {
	security, err := matrixOwnerOnlySecurityAttributes()
	if err != nil {
		return err
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.CreateDirectory(name, security)
}

// createMatrixOwnerOnlyObjectInRoot creates a new object relative to a held
// directory handle. The security descriptor is supplied to NtCreateFile,
// rather than applied after creation, so the object is owner-only from its
// first externally visible instant. Keeping the parent handle in the call
// also prevents a same-user rename of the parent path from redirecting the
// create operation.
func createMatrixOwnerOnlyObjectInRoot(parent *os.Root, name, displayPath string, directory bool) (*os.File, error) {
	if parent == nil {
		return nil, errors.New("private matrix parent is unavailable")
	}
	parentFile, err := parent.Open(".")
	if err != nil {
		return nil, err
	}
	defer parentFile.Close()

	security, err := matrixOwnerOnlySecurityAttributes()
	if err != nil {
		return nil, err
	}
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return nil, err
	}
	objectAttributes := &windows.OBJECT_ATTRIBUTES{
		Length:             uint32(unsafe.Sizeof(windows.OBJECT_ATTRIBUTES{})),
		RootDirectory:      windows.Handle(parentFile.Fd()),
		ObjectName:         objectName,
		Attributes:         windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE,
		SecurityDescriptor: security.SecurityDescriptor,
	}
	options := uint32(windows.FILE_NON_DIRECTORY_FILE)
	if directory {
		options = windows.FILE_DIRECTORY_FILE
	}
	options |= windows.FILE_SYNCHRONOUS_IO_NONALERT | windows.FILE_OPEN_REPARSE_POINT
	var handle windows.Handle
	if err := windows.NtCreateFile(
		&handle,
		windows.FILE_GENERIC_READ|windows.FILE_GENERIC_WRITE|windows.DELETE|windows.SYNCHRONIZE,
		objectAttributes,
		&windows.IO_STATUS_BLOCK{},
		nil,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		windows.FILE_CREATE,
		options,
		0,
		0,
	); err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), displayPath), nil
}

func createMatrixOwnerOnlyDirectoryInRoot(parent *os.Root, name string) error {
	file, err := createMatrixOwnerOnlyObjectInRoot(parent, name, name, true)
	if err != nil {
		return err
	}
	return file.Close()
}

func createMatrixOwnerOnlyFileInRoot(parent *os.Root, name, displayPath string) (*os.File, error) {
	return createMatrixOwnerOnlyObjectInRoot(parent, name, displayPath, false)
}

func createMatrixOwnerOnlyFile(path string) (*os.File, error) {
	security, err := matrixOwnerOnlySecurityAttributes()
	if err != nil {
		return nil, err
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		security,
		windows.CREATE_NEW,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), path), nil
}

func createMatrixPrivateScratchDir(prefix string) (string, error) {
	return createMatrixOwnerOnlyTempDir(prefix)
}

func createMatrixPrivateAttemptParent() (string, error) {
	namespace, err := createMatrixPrivateScratchDir(".asc-matrix-attempt-ns-")
	if err != nil {
		return "", err
	}
	parent := filepath.Join(namespace, "parent")
	if err := createMatrixOwnerOnlyDirectory(parent); err != nil {
		_ = os.RemoveAll(namespace)
		return "", err
	}
	return parent, nil
}

func createMatrixPrivateAttemptChild(parent *os.Root, parentPath, name string) error {
	if err := createMatrixOwnerOnlyDirectoryInRoot(parent, name); err != nil {
		return fmt.Errorf("create rooted private attempt child %s: %w", filepath.Join(parentPath, name), err)
	}
	return nil
}

func createMatrixPrivateAttemptOutputDir(workDir string) error {
	return createMatrixOwnerOnlyDirectory(filepath.Join(workDir, "output"))
}

func createMatrixPrivateAttemptOutputDirInRoot(parent *os.Root) error {
	return createMatrixOwnerOnlyDirectoryInRoot(parent, "output")
}

func createMatrixPrivateAttemptFile(path string) (*os.File, error) {
	return createMatrixOwnerOnlyFile(path)
}

func createMatrixPrivateAttemptFileInRoot(parent *os.Root, name, displayPath string) (*os.File, error) {
	return createMatrixOwnerOnlyFileInRoot(parent, name, displayPath)
}

func matrixOwnerOnlyProtectedDACL(file *os.File) bool {
	if file == nil {
		return false
	}
	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(file.Fd()),
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return false
	}
	control, _, err := descriptor.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		return false
	}
	dacl, _, err := descriptor.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 1 {
		return false
	}
	return true
}
