//go:build windows

package telemetry

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func configureDetachedProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: uint32(windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP),
		HideWindow:    true,
	}
}
