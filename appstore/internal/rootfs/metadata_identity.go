package rootfs

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
)

// fileIdentityMetadataSizeLimit bounds the amount of filesystem metadata that
// one identity capture will inspect. The retained identity stores only the
// fixed-size digests below, but the bound also prevents a provider-controlled
// xattr list or value from forcing an unbounded temporary allocation.
const fileIdentityMetadataSizeLimit int64 = 1 << 20

type fileIdentityMetadata struct {
	acl    metadataDigest
	xattrs metadataDigest
}

type metadataDigest struct {
	supported bool
	size      uint64
	sum       [sha256.Size]byte
}

func (first metadataDigest) equal(second metadataDigest) bool {
	return first.supported == second.supported &&
		first.size == second.size && first.sum == second.sum
}

func sameFileIdentityMetadata(first, second fileIdentityMetadata) bool {
	return first.acl.equal(second.acl) && first.xattrs.equal(second.xattrs)
}

func supportedEmptyMetadataDigest() metadataDigest {
	return metadataDigest{supported: true, sum: sha256.Sum256(nil)}
}

func unsupportedMetadataDigest() metadataDigest {
	return metadataDigest{}
}

func metadataDigestFromHash(hasher hash.Hash, size uint64) metadataDigest {
	var sum [sha256.Size]byte
	copy(sum[:], hasher.Sum(nil))
	return metadataDigest{supported: true, size: size, sum: sum}
}

type boundedMetadataHasher struct {
	hash hash.Hash
	size uint64
	kind string
}

func newBoundedMetadataHasher(kind string) *boundedMetadataHasher {
	return &boundedMetadataHasher{
		hash: sha256.New(),
		kind: kind,
	}
}

func (hasher *boundedMetadataHasher) append(data []byte) error {
	if hasher.size > uint64(fileIdentityMetadataSizeLimit) ||
		uint64(len(data)) > uint64(fileIdentityMetadataSizeLimit)-hasher.size {
		return fmt.Errorf("%s exceeds %d-byte identity metadata limit: %w", hasher.kind, fileIdentityMetadataSizeLimit, ErrFileIdentityDataTooLarge)
	}
	if _, err := hasher.hash.Write(data); err != nil {
		return fmt.Errorf("hash %s: %w", hasher.kind, err)
	}
	hasher.size += uint64(len(data))
	return nil
}

func (hasher *boundedMetadataHasher) appendUint64(value uint64) error {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	return hasher.append(encoded[:])
}

func (hasher *boundedMetadataHasher) digest() metadataDigest {
	return metadataDigestFromHash(hasher.hash, hasher.size)
}
