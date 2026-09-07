//go:build !darwin && !linux && !windows

package rootfs

import "os"

func copyReplacementMetadata(destination, _ *os.File, info os.FileInfo) error {
	return destination.Chmod(info.Mode().Perm())
}

func restoreReplacementMode(destination *os.File, info os.FileInfo) error {
	return destination.Chmod(info.Mode().Perm())
}
