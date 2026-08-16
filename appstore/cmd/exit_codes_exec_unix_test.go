//go:build !windows

package cmd

import (
	"errors"
	"os/exec"
	"testing"
)

func TestExitCodeFromErrorSignaledExecErrorRemainsGeneric(t *testing.T) {
	err := exec.Command("/bin/sh", "-c", "kill -KILL $$").Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %v, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != -1 {
		t.Fatalf("exec exit code = %d, want -1 for signal", exitErr.ExitCode())
	}
	if code := ExitCodeFromError(err); code != ExitError {
		t.Fatalf("mapped exit = %d, want generic %d", code, ExitError)
	}
}
