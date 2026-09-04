//go:build linux

package xcode

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Common Linux filesystem magics that compare names case-insensitively.
// FUSE and overlay mounts are left unknown so an uncertain alias cannot
// reach an artifact write.
const (
	signingLinuxMSDOSSuperMagic = 0x4d44
	signingLinuxExfatSuperMagic = 0x2011bab0
	signingLinuxNTFSSuperMagic  = 0x5346544e
	signingLinuxSMBSuperMagic   = 0x517b
	signingLinuxCIFSSuperMagic  = 0xff534d42
)

func signingLinuxFilesystemCaseInsensitive(fsType int64) bool {
	switch fsType {
	case signingLinuxMSDOSSuperMagic, signingLinuxExfatSuperMagic, signingLinuxNTFSSuperMagic, signingLinuxSMBSuperMagic, signingLinuxCIFSSuperMagic:
		return true
	default:
		return false
	}
}

// signingCaseInsensitiveVolumeFor reports the containing volume's name
// comparison semantics without creating a probe entry. The supplied path may
// not exist; the nearest existing ancestor is sufficient for statfs.
func signingCaseInsensitiveVolumeFor(path string) (caseInsensitive, known bool) {
	candidate := normalizeSigningLexicalPath(path)
	for {
		var stat unix.Statfs_t
		err := unix.Statfs(candidate, &stat)
		if err == nil {
			if signingLinuxFilesystemCaseInsensitive(int64(stat.Type)) {
				return true, true
			}
			// Unlisted filesystems may still be case-insensitive (ext4
			// casefold, FUSE, overlay). Treat them as unknown so a missing
			// PLAN.json cannot bypass alias checks.
			return false, false
		}
		if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, unix.ENOENT) {
			return false, false
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return false, false
		}
		candidate = parent
	}
}
