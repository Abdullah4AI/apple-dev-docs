//go:build !windows

package certificates

import (
	"fmt"
	"os"
)

func validateCertificateExportProtectedFile(file *os.File, info os.FileInfo, label string) error {
	if file == nil {
		return permissionErrorForCertificateExport(label)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return permissionErrorForCertificateExport(label)
	}
	// Extended ACL entries grant access independently of the mode bits, so a
	// 0600 key or password file can still be readable by another account.
	hasACL, err := certificateExportFileHasACL(file)
	if err != nil {
		return fmt.Errorf("inspect %s permissions: %w", label, err)
	}
	if hasACL {
		return fmt.Errorf("%s has an extended ACL that can grant other accounts access; remove it (chmod -N on macOS, setfacl -b on Linux)", label)
	}
	return nil
}

// prepareCertificateExportOutput protects the staged output before any PKCS#12
// bytes are written. Any extended ACL inherited from the destination directory
// is stripped and the removal is verified, and the effective permission bits
// are checked because filesystems such as FAT, exFAT, and some CIFS or FUSE
// mounts ignore or translate the requested 0600 mode. Failing closed here
// keeps the identity from being published readable by another account while
// the command reports success.
func prepareCertificateExportOutput(file *os.File) error {
	if file == nil {
		return fmt.Errorf("output permissions cannot be verified")
	}
	if err := clearCertificateExportFileACL(file); err != nil {
		return fmt.Errorf("protect output permissions: %w", err)
	}
	hasACL, err := certificateExportFileHasACL(file)
	if err != nil {
		return fmt.Errorf("verify output permissions: %w", err)
	}
	if hasACL {
		return fmt.Errorf("output retains an extended ACL after removal; the --p12-out filesystem cannot restrict this file to the owner")
	}
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("verify output permissions: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf(
			"output permissions %#o are not restricted to the owner; the --p12-out filesystem must support mode 0600",
			info.Mode().Perm(),
		)
	}
	return nil
}

func createCertificateExportStagingFile(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	return createCertificateExportStagingFilePlatform(root, name, perm)
}

func permissionErrorForCertificateExport(label string) error {
	return fmt.Errorf("%s permissions must be 0600 or more restrictive", label)
}
