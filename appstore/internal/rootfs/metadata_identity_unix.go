//go:build darwin || linux

package rootfs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"

	"golang.org/x/sys/unix"
)

func captureFileIdentityMetadata(file *os.File) (fileIdentityMetadata, error) {
	if file == nil {
		return fileIdentityMetadata{}, fmt.Errorf("capture file identity metadata: nil descriptor")
	}
	acl, err := captureFileIdentityACL(file)
	if err != nil {
		return fileIdentityMetadata{}, err
	}
	xattrs, err := captureFileIdentityXattrs(file)
	if err != nil {
		return fileIdentityMetadata{}, err
	}
	return fileIdentityMetadata{acl: acl, xattrs: xattrs}, nil
}

func captureFileIdentityXattrs(file *os.File) (metadataDigest, error) {
	fd := int(file.Fd())
	size, err := unix.Flistxattr(fd, nil)
	if err != nil {
		if metadataOperationUnsupported(err) {
			return unsupportedMetadataDigest(), nil
		}
		return metadataDigest{}, fmt.Errorf("list file identity extended attributes: %w", err)
	}
	if size < 0 {
		return metadataDigest{}, fmt.Errorf("list file identity extended attributes returned negative size %d", size)
	}
	if int64(size) > fileIdentityMetadataSizeLimit {
		return metadataDigest{}, fmt.Errorf("extended attribute names exceed %d-byte identity metadata limit: %w", fileIdentityMetadataSizeLimit, ErrFileIdentityDataTooLarge)
	}
	if size == 0 {
		return supportedEmptyMetadataDigest(), nil
	}
	namesBuffer := make([]byte, size)
	readSize, err := unix.Flistxattr(fd, namesBuffer)
	if err != nil {
		if metadataOperationUnsupported(err) {
			return unsupportedMetadataDigest(), nil
		}
		return metadataDigest{}, fmt.Errorf("read file identity extended attribute names: %w", err)
	}
	if readSize != size {
		return metadataDigest{}, fmt.Errorf("file identity extended attribute name list changed during capture: got %d bytes, want %d", readSize, size)
	}
	names, err := parseIdentityXattrNames(namesBuffer[:readSize])
	if err != nil {
		return metadataDigest{}, err
	}

	hasher := newBoundedMetadataHasher("file identity extended attributes")
	if err := hasher.append([]byte("asc-rootfs-xattrs\x00")); err != nil {
		return metadataDigest{}, err
	}
	for _, name := range names {
		valueSize, err := unix.Fgetxattr(fd, name, nil)
		if err != nil {
			if metadataOperationUnsupported(err) {
				return unsupportedMetadataDigest(), nil
			}
			return metadataDigest{}, fmt.Errorf("read file identity extended attribute %q size: %w", name, err)
		}
		if valueSize < 0 {
			return metadataDigest{}, fmt.Errorf("read file identity extended attribute %q returned negative size %d", name, valueSize)
		}
		if err := hasher.appendUint64(uint64(len(name))); err != nil {
			return metadataDigest{}, err
		}
		if err := hasher.append([]byte(name)); err != nil {
			return metadataDigest{}, err
		}
		if err := hasher.appendUint64(uint64(valueSize)); err != nil {
			return metadataDigest{}, err
		}
		if int64(valueSize) > fileIdentityMetadataSizeLimit-int64(hasher.size) {
			return metadataDigest{}, fmt.Errorf("extended attribute %q exceeds %d-byte identity metadata limit: %w", name, fileIdentityMetadataSizeLimit, ErrFileIdentityDataTooLarge)
		}
		value := make([]byte, valueSize)
		if valueSize > 0 {
			readValueSize, err := unix.Fgetxattr(fd, name, value)
			if err != nil {
				if metadataOperationUnsupported(err) {
					return unsupportedMetadataDigest(), nil
				}
				return metadataDigest{}, fmt.Errorf("read file identity extended attribute %q: %w", name, err)
			}
			if readValueSize != valueSize {
				return metadataDigest{}, fmt.Errorf("read file identity extended attribute %q returned %d bytes, want %d", name, readValueSize, valueSize)
			}
		}
		if err := hasher.append(value); err != nil {
			return metadataDigest{}, fmt.Errorf("hash file identity extended attribute %q: %w", name, err)
		}
	}
	return hasher.digest(), nil
}

func parseIdentityXattrNames(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if raw[len(raw)-1] != 0 {
		return nil, fmt.Errorf("file identity extended attribute names are not NUL terminated")
	}
	parts := bytes.Split(raw, []byte{0})
	names := make([]string, 0, len(parts)-1)
	for _, part := range parts[:len(parts)-1] {
		if len(part) == 0 {
			return nil, fmt.Errorf("file identity extended attribute names contain an empty entry")
		}
		names = append(names, string(part))
	}
	sort.Strings(names)
	for index := 1; index < len(names); index++ {
		if names[index] == names[index-1] {
			return nil, fmt.Errorf("file identity extended attribute names contain duplicate %q", names[index])
		}
	}
	return names, nil
}

func metadataOperationUnsupported(err error) bool {
	return errors.Is(err, unix.ENOTSUP) ||
		errors.Is(err, unix.EOPNOTSUPP) ||
		errors.Is(err, unix.ENOSYS)
}
