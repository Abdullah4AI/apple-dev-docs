//go:build !windows

package rootfs

import (
	"syscall"
	"testing"
)

func TestUnsupportedDirectorySyncErrorOnUnix(t *testing.T) {
	if !unsupportedDirectorySyncError(syscall.EINVAL) {
		t.Fatal("EINVAL should be treated as an unsupported directory sync")
	}
	if unsupportedDirectorySyncError(syscall.EPERM) {
		t.Fatal("EPERM should remain a directory sync failure")
	}
}
