//go:build !windows

package distribution

import "os"

func syncPreparedRootDirectory(root *os.Root, _ string) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}
