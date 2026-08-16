//go:build !darwin && !windows

package xcode

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

func TestUnixProcessGroupCleanupKeepsDurableIdentityAfterCommandExit(t *testing.T) {
	wantCleanupErr := errors.New("forced process-group cleanup failure")
	originalTerminate := terminateXcodeProcessGroupFn
	cleanupPID := 0
	terminateXcodeProcessGroupFn = func(pid int) error {
		if err := syscall.Kill(pid, 0); err != nil {
			t.Fatalf("process-group anchor was not live during cleanup: %v", err)
		}
		cleanupPID = pid
		return wantCleanupErr
	}
	t.Cleanup(func() { terminateXcodeProcessGroupFn = originalTerminate })

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestUnixProcessGroupExitHelper")
	cmd.Env = append(os.Environ(), "GO_WANT_XCODE_UNIX_PROCESS_GROUP_EXIT_HELPER=1")
	err := runXcodeCommandWithProcessGroupCleanup(cmd)
	if !errors.Is(err, wantCleanupErr) {
		t.Fatalf("error = %v, want cleanup failure", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 23 {
		t.Fatalf("error = %v, want child exit code 23", err)
	}
	if cleanupPID <= 0 || cleanupPID == cmd.Process.Pid {
		t.Fatalf("cleanup PID = %d, command PID = %d; want distinct durable anchor", cleanupPID, cmd.Process.Pid)
	}
}

func TestUnixProcessGroupExitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_XCODE_UNIX_PROCESS_GROUP_EXIT_HELPER") != "1" {
		return
	}
	os.Exit(23)
}
