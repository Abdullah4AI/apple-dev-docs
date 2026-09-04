//go:build darwin

package xcode

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// _PC_CASE_SENSITIVE is the Darwin pathconf selector for the case semantics
// of the filesystem containing a directory. It is not exported by x/sys.
const signingPathconfCaseSensitive = 11

// signingCaseInsensitiveVolumeFor reports the containing volume's name
// comparison semantics without creating a probe entry. The supplied path may
// not exist; the nearest existing ancestor is sufficient for pathconf.
// Unknown metadata is deliberately reported as unknown so callers can fail
// closed when deciding whether two missing artifact paths alias.
func signingCaseInsensitiveVolumeFor(path string) (caseInsensitive, known bool) {
	candidate := normalizeSigningLexicalPath(path)
	for {
		info, err := os.Lstat(candidate)
		if err == nil {
			if !info.IsDir() {
				candidate = filepath.Dir(candidate)
				continue
			}
			value, err := unix.Pathconf(candidate, signingPathconfCaseSensitive)
			if err != nil {
				return false, false
			}
			return value == 0, true
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, false
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return false, false
		}
		candidate = parent
	}
}
