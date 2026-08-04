package xcode

import (
	"errors"
	"os/exec"
	"time"
)

const xcodeCommandPipeWaitDelay = 250 * time.Millisecond

// runXcodeCommand bounds the time spent waiting for descendants that inherited
// the direct command's output descriptors after that command has exited.
func runXcodeCommand(cmd *exec.Cmd) error {
	cmd.WaitDelay = xcodeCommandPipeWaitDelay
	return normalizeXcodeCommandWaitError(cmd, cmd.Run())
}

func outputXcodeCommand(cmd *exec.Cmd) ([]byte, error) {
	cmd.WaitDelay = xcodeCommandPipeWaitDelay
	output, err := cmd.Output()
	return output, normalizeXcodeCommandWaitError(cmd, err)
}

func normalizeXcodeCommandWaitError(cmd *exec.Cmd, err error) error {
	if errors.Is(err, exec.ErrWaitDelay) && cmd.ProcessState != nil && cmd.ProcessState.Success() {
		return nil
	}
	return err
}
