//go:build darwin

package certificates

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	_ "unsafe"
)

// aclTypeExtended is ACL_TYPE_EXTENDED from <sys/acl.h>, the only ACL type
// macOS supports for files.
const aclTypeExtended = 0x100

// certificateExportFileHasACL reports whether the open file carries a macOS
// extended ACL. ACL entries grant access independently of the 0600 mode bits,
// so protected files are inspected through acl_get_fd_np like `ls -le` does.
func certificateExportFileHasACL(file *os.File) (bool, error) {
	aclRef, _, errno := syscallPtrCall(
		libcACLGetFdNpTrampolineAddr,
		file.Fd(),
		aclTypeExtended,
		0,
	)
	runtime.KeepAlive(file)
	if aclRef != 0 {
		_, _, _ = syscallCall(libcACLFreeTrampolineAddr, aclRef, 0, 0)
		return true, nil
	}
	if errno == 0 || errors.Is(errno, syscall.ENOENT) || errors.Is(errno, syscall.ENOTSUP) {
		return false, nil
	}
	return false, fmt.Errorf("inspect access control list: %w", errno)
}

// clearCertificateExportFileACL removes any extended ACL from the open file so
// inherited entries cannot expose the artifact being prepared. macOS does not
// implement fd-based ACL deletion (acl_delete_fd_np fails with ENOTSUP on
// APFS), so removal applies an empty ACL, which the kernel stores as no ACL.
func clearCertificateExportFileACL(file *os.File) error {
	emptyACL, _, errno := syscallPtrCall(libcACLInitTrampolineAddr, 0, 0, 0)
	if emptyACL == 0 {
		if errno == 0 {
			errno = syscall.ENOMEM
		}
		return fmt.Errorf("remove access control list: %w", errno)
	}
	defer func() {
		_, _, _ = syscallCall(libcACLFreeTrampolineAddr, emptyACL, 0, 0)
	}()

	result, _, errno := syscallCall(
		libcACLSetFdNpTrampolineAddr,
		file.Fd(),
		emptyACL,
		aclTypeExtended,
	)
	runtime.KeepAlive(file)
	if result == 0 || errors.Is(errno, syscall.ENOENT) || errors.Is(errno, syscall.ENOTSUP) {
		return nil
	}
	if errno != 0 {
		return fmt.Errorf("remove access control list: %w", errno)
	}
	return fmt.Errorf("remove access control list: unexpected result %d", int64(result))
}

var (
	libcACLGetFdNpTrampolineAddr uintptr
	libcACLFreeTrampolineAddr    uintptr
	libcACLInitTrampolineAddr    uintptr
	libcACLSetFdNpTrampolineAddr uintptr
)

//go:cgo_import_dynamic libc_acl_get_fd_np acl_get_fd_np "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_acl_free acl_free "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_acl_init acl_init "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_acl_set_fd_np acl_set_fd_np "/usr/lib/libSystem.B.dylib"

//go:linkname syscallCall syscall.syscall
func syscallCall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

// syscallPtrCall is the libc-call helper for functions that return a pointer
// and report failure with a NULL result plus errno, such as acl_get_fd_np.
//
//go:linkname syscallPtrCall syscall.syscallPtr
func syscallPtrCall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)
