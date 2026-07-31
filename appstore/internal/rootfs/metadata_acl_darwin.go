//go:build darwin

package rootfs

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	_ "unsafe"
)

const copyfileACL = 1 << 0

func copyAccessControlList(destination, source *os.File) error {
	_, _, errno := syscall6(
		libcFcopyfileTrampolineAddr,
		source.Fd(),
		destination.Fd(),
		0,
		copyfileACL,
		0,
		0,
	)
	runtime.KeepAlive(source)
	runtime.KeepAlive(destination)
	if errno == 0 {
		return nil
	}
	if errors.Is(errno, syscall.ENOTSUP) {
		return nil
	}
	return fmt.Errorf("preserve replacement access control list: %w", errno)
}

var libcFcopyfileTrampolineAddr uintptr

//go:cgo_import_dynamic libc_fcopyfile fcopyfile "/usr/lib/libSystem.B.dylib"

//go:linkname syscall6 syscall.syscall6
func syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)
