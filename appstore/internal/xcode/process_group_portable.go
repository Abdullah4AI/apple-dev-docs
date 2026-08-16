//go:build windows

package xcode

import (
	"fmt"
	"os/exec"
)

func runXcodeCommandWithProcessGroupCleanup(*exec.Cmd) error {
	return fmt.Errorf("release-testing process-group cleanup is supported on macOS only")
}
