//go:build windows

package web

import "golang.org/x/sys/windows"

func platformSessionLockRoot() string {
	dir, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return ""
	}
	return dir
}

func platformSessionLockDirName() string { return "asc-web-session-locks" }
