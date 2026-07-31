//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly && !windows

package install

import (
	"errors"
	"os"
	"sync"
)

var (
	errInstallLockHeld  = errors.New("install lock is held")
	fallbackInstallLock sync.Mutex
	fallbackLockHeld    bool
)

func lockInstallFile(_ *os.File) error {
	fallbackInstallLock.Lock()
	defer fallbackInstallLock.Unlock()
	if fallbackLockHeld {
		return errInstallLockHeld
	}
	fallbackLockHeld = true
	return nil
}

func unlockInstallFile(_ *os.File) error {
	fallbackInstallLock.Lock()
	fallbackLockHeld = false
	fallbackInstallLock.Unlock()
	return nil
}
