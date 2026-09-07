package rootfs

import (
	"errors"
	"testing"
)

func TestMetadataHasherRejectsOversizeInput(t *testing.T) {
	hasher := newBoundedMetadataHasher("test metadata")
	oversize := make([]byte, fileIdentityMetadataSizeLimit+1)
	if err := hasher.append(oversize); !errors.Is(err, ErrFileIdentityDataTooLarge) {
		t.Fatalf("boundedMetadataHasher.append() error = %v, want ErrFileIdentityDataTooLarge", err)
	}
}

func TestSameFileIdentityMetadataComparesACLAndXattrs(t *testing.T) {
	base := fileIdentityMetadata{
		acl:    metadataDigest{supported: true, size: 1, sum: [32]byte{1}},
		xattrs: metadataDigest{supported: true, size: 1, sum: [32]byte{2}},
	}
	if !sameFileIdentityMetadata(base, base) {
		t.Fatal("identical metadata snapshots should compare equal")
	}
	changedACL := base
	changedACL.acl.sum[0]++
	if sameFileIdentityMetadata(base, changedACL) {
		t.Fatal("ACL drift should compare unequal")
	}
	changedXattrs := base
	changedXattrs.xattrs.sum[0]++
	if sameFileIdentityMetadata(base, changedXattrs) {
		t.Fatal("xattr drift should compare unequal")
	}
	unsupported := base
	unsupported.xattrs.supported = false
	if sameFileIdentityMetadata(base, unsupported) {
		t.Fatal("supported to unsupported metadata transition should compare unequal")
	}
}
