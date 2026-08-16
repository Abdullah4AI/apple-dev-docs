package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltBinaryOrdinaryExecFailureRemainsGeneric(t *testing.T) {
	binary := buildASCBlackboxBinary(t)
	cmd := exec.Command(
		binary,
		"signing", "sync", "pull",
		"--repo", "file:///definitely/not/an/asc/repo",
		"--password", "test-only",
		"--output-dir", filepath.Join(t.TempDir(), "output"),
		"--output", "json",
	)
	cmd.Env = append(os.Environ(), "ASC_BYPASS_KEYCHAIN=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %v, want nonzero built-binary exit", err)
	}
	if exitErr.ExitCode() != ExitError {
		t.Fatalf("exit = %d, want %d; stderr=%q", exitErr.ExitCode(), ExitError, stderr.String())
	}
	if !strings.Contains(stderr.String(), "git clone") {
		t.Fatalf("stderr = %q, want git clone failure", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}
