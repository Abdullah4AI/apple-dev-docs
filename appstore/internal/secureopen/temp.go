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
	return CreateTempNoFollowInRootWithCreator(root, dir, pattern, perm, nil)
}

// CreateTempNoFollowInRootWithCreator creates a temporary file with a
// generated name beneath root. The creator may provide platform-specific
// creation security, but receives only the generated root-relative name and
// must return a handle for that exact file. The helper still verifies the
// resulting identity and no-follow contract before returning it.
func CreateTempNoFollowInRootWithCreator(root *os.Root, dir string, pattern string, perm os.FileMode, creator func(*os.Root, string, os.FileMode) (*os.File, error)) (*os.File, string, error) {
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
		create := creator
		if create == nil {
			create = OpenNewFileNoFollowInRoot
		}
		file, err := create(root, name, perm)
		if err == nil {
			if file == nil {
				return nil, name, fmt.Errorf("temporary creator returned a nil file")
			}
			if verifyErr := verifyRootOpenedPath(root, name, file, nil); verifyErr != nil {
				_ = file.Close()
				return nil, name, verifyErr
			}
			return file, name, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return nil, name, err
	}

	return nil, "", fmt.Errorf("failed to create temporary file in %q beneath root %q", dir, root.Name())
}
