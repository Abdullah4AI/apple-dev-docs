//go:build windows

package distribution

import "os"

func syncPreparedRootDirectory(_ *os.Root, _ string) error {
	// Windows directory handles cannot be portably flushed through os.Root.
	// Each staged file is flushed before the atomic no-replace directory rename.
	return nil
}
