//go:build windows

package signing

import "os/exec"

// Windows process-tree cancellation uses the platform Git process contract;
// the command context still terminates and waits for the Git child.
func configureGitProcess(*exec.Cmd) {}
