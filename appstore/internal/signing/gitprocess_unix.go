//go:build !windows

package signing

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// configureGitProcess gives each Git invocation its own process group. Git
// commonly starts transport helpers such as SSH; cancelling only the Git
// parent can otherwise leave those children running after the command has
// returned.
func configureGitProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}
