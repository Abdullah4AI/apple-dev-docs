package secureopen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrRenameNoReplaceUnsupported reports that the platform or filesystem does
// not provide the requested atomic no-replace rename operation.
var ErrRenameNoReplaceUnsupported = errors.New("atomic no-replace rename unsupported")

func validateRenameNoReplaceNames(oldName, newName string) error {
	for _, name := range []string{oldName, newName} {
		if name == "" || name == "." || name == ".." || filepath.IsAbs(name) || filepath.VolumeName(name) != "" || filepath.Base(name) != name {
			return fmt.Errorf("invalid root-relative file name %q", name)
		}
	}
	return nil
}

func renameNoReplaceError(op, oldName, newName string, err error) error {
	if err == nil {
		return nil
	}
	return &os.LinkError{Op: op, Old: oldName, New: newName, Err: err}
}

func unsupportedRenameNoReplaceError(op, oldName, newName string, err error) error {
	if err == nil {
		err = ErrRenameNoReplaceUnsupported
	} else {
		err = errors.Join(ErrRenameNoReplaceUnsupported, err)
	}
	return renameNoReplaceError(op, oldName, newName, err)
}
