//go:build !darwin && !linux && !windows

package secureopen

import "os"

// RenameNoReplaceInRoot reports that atomic no-replace rename is unavailable
// on this platform.
func RenameNoReplaceInRoot(root *os.Root, oldName, newName string) error {
	const op = "rename"
	if err := validateRenameNoReplaceNames(oldName, newName); err != nil {
		return renameNoReplaceError(op, oldName, newName, err)
	}
	return unsupportedRenameNoReplaceError(op, oldName, newName, nil)
}
