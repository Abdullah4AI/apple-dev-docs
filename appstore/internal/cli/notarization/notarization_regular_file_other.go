//go:build !darwin && !linux

package notarization

import "os"

func newStaplerRegularFileAccess(string, string, *os.File) (*staplerRegularFileAccess, error) {
	return nil, errStaplerRegularFileUnsupported
}
