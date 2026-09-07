//go:build darwin

package rootfs

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const darwinACLTypeExtended = 0x100

func captureFileIdentityACL(file *os.File) (metadataDigest, error) {
	aclRef, _, errno := metadataSyscallPtrCall(
		libcACLGetFdNpIdentityTrampolineAddr,
		file.Fd(),
		darwinACLTypeExtended,
		0,
	)
	runtime.KeepAlive(file)
	if aclRef == 0 {
		if errno == 0 || errors.Is(errno, syscall.ENOENT) {
			return supportedEmptyMetadataDigest(), nil
		}
		if metadataOperationUnsupported(errno) {
			return unsupportedMetadataDigest(), nil
		}
		return metadataDigest{}, fmt.Errorf("read file identity access control list: %w", errno)
	}
	defer func() {
		_, _, _ = syscall6(
			libcACLFreeIdentityTrampolineAddr,
			aclRef,
			0,
			0,
			0,
			0,
			0,
		)
	}()

	rawSize, _, errno := syscall6(
		libcACLSizeIdentityTrampolineAddr,
		aclRef,
		0,
		0,
		0,
		0,
		0,
	)
	size := int64(rawSize)
	if size < 0 {
		if errno == 0 {
			errno = syscall.EINVAL
		}
		return metadataDigest{}, fmt.Errorf("inspect file identity access control list size: %w", errno)
	}
	if size > fileIdentityMetadataSizeLimit {
		return metadataDigest{}, fmt.Errorf("access control list exceeds %d-byte identity metadata limit: %w", fileIdentityMetadataSizeLimit, ErrFileIdentityDataTooLarge)
	}
	if size == 0 {
		return supportedEmptyMetadataDigest(), nil
	}
	serialized := make([]byte, size)
	copiedRaw, _, errno := syscall6(
		libcACLCopyExtIdentityTrampolineAddr,
		uintptr(unsafe.Pointer(&serialized[0])),
		aclRef,
		uintptr(size),
		0,
		0,
		0,
	)
	copied := int64(copiedRaw)
	if copied < 0 {
		if errno == 0 {
			errno = syscall.EINVAL
		}
		return metadataDigest{}, fmt.Errorf("serialize file identity access control list: %w", errno)
	}
	if copied != size {
		return metadataDigest{}, fmt.Errorf("serialize file identity access control list returned %d bytes, want %d", copied, size)
	}
	hasher := newBoundedMetadataHasher("file identity access control list")
	if err := hasher.append([]byte("asc-rootfs-darwin-acl\x00")); err != nil {
		return metadataDigest{}, err
	}
	if err := hasher.append(serialized); err != nil {
		return metadataDigest{}, err
	}
	return hasher.digest(), nil
}

var (
	libcACLGetFdNpIdentityTrampolineAddr uintptr
	libcACLFreeIdentityTrampolineAddr    uintptr
	libcACLSizeIdentityTrampolineAddr    uintptr
	libcACLCopyExtIdentityTrampolineAddr uintptr
)

//go:cgo_import_dynamic libc_acl_get_fd_np_identity acl_get_fd_np "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_acl_free_identity acl_free "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_acl_size_identity acl_size "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_acl_copy_ext_identity acl_copy_ext "/usr/lib/libSystem.B.dylib"

//go:linkname metadataSyscallPtrCall syscall.syscallPtr
func metadataSyscallPtrCall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)
