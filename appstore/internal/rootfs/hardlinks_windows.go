//go:build windows

package rootfs

import (
	"os"

	"golang.org/x/sys/windows"
)

func hasMultipleHardLinks(file *os.File, _ os.FileInfo) (bool, error) {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); err != nil {
		return false, err
	}
	return info.NumberOfLinks > 1, nil
}
