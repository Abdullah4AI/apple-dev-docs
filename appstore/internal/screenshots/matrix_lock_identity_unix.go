//go:build aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package screenshots

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func matrixStableFilesystemIdentity(root *os.Root) (string, error) {
	if root == nil {
		return "", errors.New("opened output root is required")
	}
	info, err := root.Stat(".")
	if err != nil {
		return "", err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return "", errors.New("opened output root has no stable filesystem identity")
	}
	return fmt.Sprintf("unix:%d:%d", stat.Dev, stat.Ino), nil
}
