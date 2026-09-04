//go:build linux

package xcode

import "testing"

func TestSigningLinuxFilesystemCaseInsensitive(t *testing.T) {
	if !signingLinuxFilesystemCaseInsensitive(signingLinuxMSDOSSuperMagic) {
		t.Fatal("vfat should be case-insensitive")
	}
	if signingLinuxFilesystemCaseInsensitive(0x9123683e) { // BTRFS
		t.Fatal("btrfs should not be classified as a known insensitive volume")
	}
}
