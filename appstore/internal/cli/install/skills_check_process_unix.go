//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package install

import (
	"os/exec"
	"syscall"
)

func configureDetachedSkillsCheckProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
