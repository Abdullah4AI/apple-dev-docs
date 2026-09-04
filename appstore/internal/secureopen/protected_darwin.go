//go:build darwin

package secureopen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	darwinKauthFilesecMagic     = 0x012cc16d
	darwinKauthFileSecNoInherit = 1 << 17
	darwinKauthUIDNone          = ^uint32(0) - 100
	darwinKauthGIDNone          = ^uint32(0) - 100
	darwinMaxPathLength         = 1024
)

// darwinNoInheritFileSec is the fixed-size header for a kauth_filesec with an
// explicitly empty ACL. An empty ACL marked KAUTH_FILESEC_NO_INHERIT tells the
// kernel not to compose inheritable entries from the parent while creating the
// file.
type darwinNoInheritFileSec struct {
	magic      uint32
	owner      [16]byte
	group      [16]byte
	entryCount uint32
	aclFlags   uint32
}

var libcFcntlTrampolineAddr uintptr

//go:cgo_import_dynamic libc_fcntl fcntl "/usr/lib/libSystem.B.dylib"

//go:linkname syscallCall syscall.syscall
func syscallCall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

// darwinOpenedDirectoryPath returns the kernel-resolved absolute name for an
// already-open directory. Using the held descriptor avoids recreating a
// /tmp-/private or /var-/private alias from Root.Name before the protected
// create. The subsequent rooted identity check remains mandatory because
// open_extended itself accepts a path rather than a directory descriptor.
func darwinOpenedDirectoryPath(directory *os.File) (string, error) {
	if directory == nil {
		return "", fmt.Errorf("missing opened parent directory")
	}
	var path [darwinMaxPathLength]byte
	raw, err := directory.SyscallConn()
	if err != nil {
		return "", err
	}
	var syscallErr error
	if err := raw.Control(func(fd uintptr) {
		_, _, errno := syscallCall(
			libcFcntlTrampolineAddr,
			fd,
			uintptr(unix.F_GETPATH),
			uintptr(unsafe.Pointer(&path[0])),
		)
		if errno != 0 {
			syscallErr = errno
		}
	}); err != nil {
		return "", err
	}
	runtime.KeepAlive(directory)
	if syscallErr != nil {
		return "", syscallErr
	}
	pathEnd := bytes.IndexByte(path[:], 0)
	if pathEnd <= 0 {
		return "", fmt.Errorf("opened parent directory returned an invalid path")
	}
	return string(path[:pathEnd]), nil
}

// OpenNewFileNoFollowInRootNoInherit creates a new owner-only file beneath a
// rooted directory on Darwin. Unlike a regular openat(O_CREAT|O_EXCL), the
// initial ACL is supplied to the kernel as part of creation, so inheritable
// parent ACL entries are never visible on the new inode.
func OpenNewFileNoFollowInRootNoInherit(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	return openNewFileNoFollowInRootNoInherit(root, name, perm, nil)
}

func openNewFileNoFollowInRootNoInherit(root *os.Root, name string, perm os.FileMode, observer func(*os.File)) (*os.File, error) {
	if root == nil {
		return nil, fmt.Errorf("root is nil")
	}
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || filepath.Clean(name) != name {
		return nil, fmt.Errorf("invalid root-relative file name %q", name)
	}
	if perm&^0o777 != 0 {
		return nil, fmt.Errorf("unsupported file mode %#o", perm)
	}
	if perm&0o077 != 0 {
		return nil, fmt.Errorf("protected file mode must be owner-only")
	}

	parent, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	path, err := darwinOpenedDirectoryPath(parent)
	if err != nil {
		return nil, err
	}
	path = filepath.Join(path, name)
	pathPointer, err := syscall.BytePtrFromString(path)
	if err != nil {
		return nil, err
	}
	security := darwinNoInheritFileSec{
		magic:      darwinKauthFilesecMagic,
		entryCount: 0,
		aclFlags:   darwinKauthFileSecNoInherit,
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL | unix.O_NOFOLLOW
	// libSystem does not export open_extended on current Darwin, so the
	// kernel syscall is the supported way to attach a filesec at creation.
	rawFD, _, errno := syscall.Syscall6( //nolint:staticcheck
		unix.SYS_OPEN_EXTENDED, //nolint:staticcheck
		uintptr(unsafe.Pointer(pathPointer)),
		uintptr(flags),
		uintptr(darwinKauthUIDNone),
		uintptr(darwinKauthGIDNone),
		uintptr(perm.Perm()),
		uintptr(unsafe.Pointer(&security)),
	)
	runtime.KeepAlive(pathPointer)
	runtime.KeepAlive(&security)
	if errno != 0 {
		return nil, errno
	}

	file := os.NewFile(rawFD, path)
	if file == nil {
		_ = syscall.Close(int(rawFD))
		return nil, fmt.Errorf("cannot wrap created descriptor")
	}
	if observer != nil {
		observer(file)
	}
	if err := verifyRootOpenedPath(root, name, file, nil); err != nil {
		return nil, closeAfterVerificationFailure(file, err)
	}
	return file, nil
}
