//go:build !windows && !darwin && !linux

package certificates

import "os"

// Extended-ACL inspection is implemented for the platforms this project
// supports (darwin, linux, windows). Other Unix-like platforms keep the plain
// permission-bit policy, matching internal/rootfs, which preserves ACL
// metadata only on darwin and linux.
func certificateExportFileHasACL(_ *os.File) (bool, error) {
	return false, nil
}

func clearCertificateExportFileACL(_ *os.File) error {
	return nil
}
