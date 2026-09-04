//go:build !aix && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !solaris && !windows

package screenshots

import (
	"errors"
	"os"
)

func matrixStableFilesystemIdentity(_ *os.Root) (string, error) {
	return "", errors.New("opened output root has no supported stable filesystem identity")
}
