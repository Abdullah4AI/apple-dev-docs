//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package signing

import (
	"fmt"
	"os"
	"syscall"
)

func validateReconcileProtectedFilePlatform(_ *os.File, info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint64(stat.Uid) != uint64(os.Geteuid()) {
		return fmt.Errorf("protected input must be owned by the current user")
	}
	if stat.Nlink != 1 {
		return fmt.Errorf("protected input must not have multiple hard links")
	}
	return nil
}
