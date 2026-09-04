//go:build windows

package screenshots

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func matrixStableFilesystemIdentity(root *os.Root) (string, error) {
	if root == nil {
		return "", errors.New("opened output root is required")
	}
	file, err := root.Open(".")
	if err != nil {
		return "", err
	}
	defer file.Close()
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); err != nil {
		return "", err
	}
	return fmt.Sprintf("windows:%d:%d:%d", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow), nil
}
