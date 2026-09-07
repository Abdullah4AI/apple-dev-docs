//go:build linux

package rootfs

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

const linuxPOSIXACLAccessIdentityAttribute = "system.posix_acl_access"

func captureFileIdentityACL(file *os.File) (metadataDigest, error) {
	fd := int(file.Fd())
	size, err := unix.Fgetxattr(fd, linuxPOSIXACLAccessIdentityAttribute, nil)
	if err != nil {
		if errors.Is(err, unix.ENODATA) {
			return supportedEmptyMetadataDigest(), nil
		}
		if metadataOperationUnsupported(err) {
			return unsupportedMetadataDigest(), nil
		}
		return metadataDigest{}, fmt.Errorf("read file identity POSIX access ACL size: %w", err)
	}
	if size < 0 {
		return metadataDigest{}, fmt.Errorf("read file identity POSIX access ACL returned negative size %d", size)
	}
	if int64(size) > fileIdentityMetadataSizeLimit {
		return metadataDigest{}, fmt.Errorf("POSIX access ACL exceeds %d-byte identity metadata limit: %w", fileIdentityMetadataSizeLimit, ErrFileIdentityDataTooLarge)
	}
	value := make([]byte, size)
	if size > 0 {
		readSize, err := unix.Fgetxattr(fd, linuxPOSIXACLAccessIdentityAttribute, value)
		if err != nil {
			if metadataOperationUnsupported(err) {
				return unsupportedMetadataDigest(), nil
			}
			return metadataDigest{}, fmt.Errorf("read file identity POSIX access ACL: %w", err)
		}
		if readSize != size {
			return metadataDigest{}, fmt.Errorf("read file identity POSIX access ACL returned %d bytes, want %d", readSize, size)
		}
	}
	hasher := newBoundedMetadataHasher("file identity POSIX access ACL")
	if err := hasher.append([]byte("asc-rootfs-posix-acl-access\x00")); err != nil {
		return metadataDigest{}, err
	}
	if err := hasher.append(value); err != nil {
		return metadataDigest{}, err
	}
	return hasher.digest(), nil
}
