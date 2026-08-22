package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSystemStatusInvalidOutputWithBuiltBinary(t *testing.T) {
	binaryPath := buildASCBlackboxBinary(t)
	command := exec.Command(binaryPath, "system-status", "--output", "yaml")
	command.Env = append(os.Environ(), "ASC_BYPASS_KEYCHAIN=1")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != ExitUsage {
		t.Fatalf("exit error = %v, want exit %d", err, ExitUsage)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `Error: --output must be one of: json, table, markdown (got "yaml")`) {
		t.Fatalf("stderr = %q, want invalid-output diagnostic", stderr.String())
	}
}
