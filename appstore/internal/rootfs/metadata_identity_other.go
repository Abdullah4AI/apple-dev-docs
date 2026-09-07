//go:build !darwin && !linux

package rootfs

import "os"

// Other platforms do not expose a portable ACL/xattr descriptor API here. The
// strict identity mutation path is already unavailable on Windows, and the
// zero snapshot preserves the existing read-only identity behavior elsewhere.
func captureFileIdentityMetadata(_ *os.File) (fileIdentityMetadata, error) {
	return fileIdentityMetadata{}, nil
}
