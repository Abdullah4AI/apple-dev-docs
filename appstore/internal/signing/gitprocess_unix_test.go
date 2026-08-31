//go:build !windows

package signing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	gitProcessTestTimeout       = 2 * time.Second
	gitProcessTestStartupWindow = 3 * time.Second
)

func TestGitStoreCloneCancellationTerminatesProcessGroupAndWaits(t *testing.T) {
	startedPath := filepath.Join(t.TempDir(), "clone-started")
	childPIDPath := filepath.Join(t.TempDir(), "clone-child-pid")
	environmentPath := filepath.Join(t.TempDir(), "clone-environment")
	successPath := filepath.Join(t.TempDir(), "clone-success")
	configureGitHelperProcess(t, "clone", startedPath, childPIDPath, environmentPath, successPath)

	store := &GitStore{
		RepoURL:  "ssh://example.invalid/signing.git",
		LocalDir: filepath.Join(t.TempDir(), "clone"),
		Branch:   "main",
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitProcessTestTimeout)
	defer cancel()

	started := make(chan struct{})
	go waitForGitHelperFile(t, startedPath, started)
	startedAt := time.Now()
	done := make(chan error, 1)
	go func() { done <- store.Clone(ctx, true) }()
	select {
	case <-started:
	case <-time.After(gitProcessTestStartupWindow):
		t.Fatal("Git clone helper did not start")
	}

	childPID := readGitHelperPID(t, childPIDPath)

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled Git clone unexpectedly succeeded")
		}
	case <-time.After(gitProcessTestStartupWindow):
		t.Fatal("Git clone did not return after context cancellation")
	}
	if ctx.Err() == nil {
		t.Fatal("Git clone returned before its context was canceled")
	}
	if elapsed := time.Since(startedAt); elapsed > gitProcessTestStartupWindow+gitProcessTestTimeout {
		t.Fatalf("canceled Git clone took %v, want bounded termination", elapsed)
	}
	assertGitHelperProcessExited(t, childPID)
	assertGitHelperEnvironmentWasRedacted(t, environmentPath)
	assertGitHelperDidNotReportSuccess(t, successPath)
	if err := store.Cleanup(); err != nil {
		t.Fatalf("cleanup canceled clone: %v", err)
	}
	if _, err := os.Stat(store.LocalDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled clone left temporary directory: %v", err)
	}
}

func TestGitStorePushCancellationTerminatesProcessGroupAndWaits(t *testing.T) {
	startedPath := filepath.Join(t.TempDir(), "push-started")
	childPIDPath := filepath.Join(t.TempDir(), "push-child-pid")
	environmentPath := filepath.Join(t.TempDir(), "push-environment")
	successPath := filepath.Join(t.TempDir(), "push-success")
	configureGitHelperProcess(t, "push", startedPath, childPIDPath, environmentPath, successPath)

	store := &GitStore{LocalDir: t.TempDir(), Branch: "main"}
	ctx, cancel := context.WithTimeout(context.Background(), gitProcessTestTimeout)
	defer cancel()

	started := make(chan struct{})
	go waitForGitHelperFile(t, startedPath, started)
	done := make(chan error, 1)
	go func() { done <- store.CommitAndPush(ctx, "test signing update") }()
	select {
	case <-started:
	case <-time.After(gitProcessTestStartupWindow):
		t.Fatal("Git push helper did not start")
	}
	childPID := readGitHelperPID(t, childPIDPath)

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled Git push unexpectedly succeeded")
		}
	case <-time.After(gitProcessTestStartupWindow):
		t.Fatal("Git push did not return after context cancellation")
	}
	if ctx.Err() == nil {
		t.Fatal("Git push returned before its context was canceled")
	}
	assertGitHelperProcessExited(t, childPID)
	assertGitHelperEnvironmentWasRedacted(t, environmentPath)
	assertGitHelperDidNotReportSuccess(t, successPath)
	if err := store.Cleanup(); err != nil {
		t.Fatalf("cleanup canceled push: %v", err)
	}
	if _, err := os.Stat(store.LocalDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled push left temporary directory: %v", err)
	}
}

// TestGitStoreGitHelperProcess is launched by the fake Git executable below.
// It deliberately starts a child so cancellation tests verify process-group
// cleanup rather than merely observing the direct Git process exit.
func TestGitStoreGitHelperProcess(t *testing.T) {
	if os.Getenv("ASC_GIT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	gitArgs := argsAfterTestSeparator(args)
	gitOperation := ""
	if len(gitArgs) > 0 {
		gitOperation = gitArgs[0]
	}
	writeGitHelperEnvironment(os.Getenv("ASC_GIT_HELPER_ENV_CAPTURE"))

	if gitOperation == os.Getenv("ASC_GIT_HELPER_BLOCK_OPERATION") {
		startedPath := os.Getenv("ASC_GIT_HELPER_STARTED")
		childPIDPath := os.Getenv("ASC_GIT_HELPER_CHILD_PID")
		if err := os.WriteFile(startedPath, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			os.Exit(2)
		}
		child := exec.Command("sleep", "30")
		if err := child.Start(); err != nil {
			os.Exit(3)
		}
		if err := os.WriteFile(childPIDPath, []byte(strconv.Itoa(child.Process.Pid)), 0o600); err != nil {
			os.Exit(4)
		}
		time.Sleep(30 * time.Second)
		if err := os.WriteFile(os.Getenv("ASC_GIT_HELPER_SUCCESS"), []byte("success"), 0o600); err != nil {
			os.Exit(6)
		}
	}

	if gitOperation == "clone" && len(gitArgs) > 0 {
		cloneDestination := gitArgs[len(gitArgs)-1]
		if err := os.MkdirAll(cloneDestination, 0o755); err != nil {
			os.Exit(5)
		}
	}

	if gitOperation == "status" {
		_, _ = fmt.Fprintln(os.Stdout, " M signing-artifact")
	}
	os.Exit(0)
}

func configureGitHelperProcess(t *testing.T, operation, startedPath, childPIDPath, environmentPath, successPath string) {
	t.Helper()
	binDir := t.TempDir()
	writeTestExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
set -eu
if [ "${ASC_GIT_HELPER_BLOCK_OPERATION}" != "${1:-}" ]; then
  if [ "${1:-}" = "status" ]; then
    printf '%s\n' ' M signing-artifact'
  fi
  exit 0
fi
exec "$ASC_GIT_HELPER_BINARY" -test.run=TestGitStoreGitHelperProcess -- "$@"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ASC_GIT_HELPER_BINARY", os.Args[0])
	t.Setenv("ASC_GIT_HELPER_PROCESS", "1")
	t.Setenv("ASC_GIT_HELPER_BLOCK_OPERATION", operation)
	t.Setenv("ASC_GIT_HELPER_STARTED", startedPath)
	t.Setenv("ASC_GIT_HELPER_CHILD_PID", childPIDPath)
	t.Setenv("ASC_GIT_HELPER_ENV_CAPTURE", environmentPath)
	t.Setenv("ASC_GIT_HELPER_SUCCESS", successPath)
	t.Setenv("ASC_SIGNING_SYNC_PASSWORD", "must-not-reach-git")
	t.Setenv("ASC_MATCH_PASSWORD", "legacy-must-not-reach-git")
	// A non-empty override avoids the local SSH configuration probe; the fake
	// executable is the transport boundary under test.
	t.Setenv("GIT_SSH_COMMAND", "true")
}

func argsAfterTestSeparator(args []string) []string {
	for index, arg := range args {
		if arg == "--" && index+1 < len(args) {
			return args[index+1:]
		}
	}
	return nil
}

func writeGitHelperEnvironment(path string) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(os.Getenv("ASC_SIGNING_SYNC_PASSWORD")+"\n"+os.Getenv("ASC_MATCH_PASSWORD")), 0o600)
}

func waitForGitHelperFile(t *testing.T, path string, ready chan<- struct{}) {
	t.Helper()
	deadline := time.Now().Add(gitProcessTestStartupWindow)
	for {
		if data, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(data)) != "" {
			close(ready)
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func readGitHelperPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(gitProcessTestStartupWindow)
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if parseErr != nil {
				t.Fatalf("parse helper PID %q: %v", data, parseErr)
			}
			return pid
		}
		if time.Now().After(deadline) {
			t.Fatalf("read helper PID %s: %v", path, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func assertGitHelperProcessExited(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(gitProcessTestStartupWindow)
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if err != nil {
			t.Fatalf("probe helper process %d: %v", pid, err)
		}
		if output, psErr := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output(); psErr == nil && strings.Contains(string(output), "Z") {
			return
		}
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("helper child process %d survived cancellation", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertGitHelperEnvironmentWasRedacted(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Git helper environment: %v", err)
	}
	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("Git helper received signing password environment: %q", data)
	}
}

func assertGitHelperDidNotReportSuccess(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Git helper reported success after deadline: %v", err)
	}
}
