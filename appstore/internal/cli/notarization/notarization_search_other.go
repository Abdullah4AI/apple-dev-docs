//go:build !darwin && !linux && !windows

package notarization

import (
	"errors"
	"os"
)

// Platforms without an audited search-only, no-follow open fail closed for a
// raw lexical-parent traversal. Ordinary paths without an erasable component
// are unaffected.
func openStaplerSearchableDirectoryNoFollow(path string) (*os.File, error) {
	return nil, errors.New("search-only lexical directory validation is unsupported on this platform")
}
