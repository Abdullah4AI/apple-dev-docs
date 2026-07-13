//go:build windows

package install

import (
	"os/exec"
	"syscall"
)

const (
	detachedProcessCreationFlag = 0x00000008
	newProcessGroupCreationFlag = 0x00000200
)

func configureDetachedSkillsCheckProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcessCreationFlag | newProcessGroupCreationFlag,
		HideWindow:    true,
	}
}
