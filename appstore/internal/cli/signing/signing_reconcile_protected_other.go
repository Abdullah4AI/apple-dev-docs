//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly && !windows

package signing

import (
	"fmt"
	"os"
)

func validateReconcileProtectedFilePlatform(_ *os.File, _ os.FileInfo) error {
	return fmt.Errorf("protected input ownership and link count cannot be verified on this platform")
}
