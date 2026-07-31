package secureopen

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateTempNoFollow creates a new temporary file in dir using an unpredictable
// name derived from crypto/rand, an exclusive create, and no-follow semantics.
//
// The pattern follows os.CreateTemp semantics: the last "*" is replaced with
// random text, or random text is appended when the pattern has no "*".
func CreateTempNoFollow(dir string, pattern string, perm os.FileMode) (*os.File, error) {
	prefix := pattern
	suffix := ""
	if idx := strings.LastIndex(pattern, "*"); idx != -1 {
		prefix = pattern[:idx]
		suffix = pattern[idx+1:]
	}

	const maxAttempts = 10_000
	var randBytes [12]byte
	for range maxAttempts {
		if _, err := rand.Read(randBytes[:]); err != nil {
			return nil, err
		}

		name := prefix + hex.EncodeToString(randBytes[:]) + suffix
		file, err := OpenNewFileNoFollow(filepath.Join(dir, name), perm)
		if err == nil {
			return file, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return nil, err
	}

	return nil, fmt.Errorf("failed to create temporary file in %q", dir)
}

// CreateTempNoFollowInRoot creates a new temporary file beneath root and
// returns both the file and its root-relative name. Parent resolution stays
// anchored to root even if a directory is concurrently replaced by a symlink.
func CreateTempNoFollowInRoot(root *os.Root, dir string, pattern string, perm os.FileMode) (*os.File, string, error) {
	prefix := pattern
	suffix := ""
	if idx := strings.LastIndex(pattern, "*"); idx != -1 {
		prefix = pattern[:idx]
		suffix = pattern[idx+1:]
	}

	const maxAttempts = 10_000
	var randBytes [12]byte
	for range maxAttempts {
		if _, err := rand.Read(randBytes[:]); err != nil {
			return nil, "", err
		}

		name := filepath.Join(dir, prefix+hex.EncodeToString(randBytes[:])+suffix)
		file, err := OpenNewFileNoFollowInRoot(root, name, perm)
		if err == nil {
			return file, name, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return nil, "", err
	}

	return nil, "", fmt.Errorf("failed to create temporary file in %q beneath root %q", dir, root.Name())
}
