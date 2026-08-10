//go:build !windows

package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRun_CancellationTerminatesProcessTree(t *testing.T) {
	dir := t.TempDir()
	pidPath := dir + "/child.pid"
	def, _ := loadWorkflowForRetryTest(t, fmt.Sprintf(`{
		"env": {"PID_PATH": %q},
		"workflows": {
			"main": {"steps": [{
				"name": "tree",
				"run": "sleep 10 & child=$!; printf '%%s' \"$child\" > \"$PID_PATH\"; wait \"$child\"",
				"timeout": "1h"
			}]}
		}
	}`, pidPath))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pidReady := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			data, readErr := os.ReadFile(pidPath)
			if readErr == nil && strings.TrimSpace(string(data)) != "" {
				pidReady <- nil
				cancel()
				return
			}
			select {
			case <-ctx.Done():
				pidReady <- fmt.Errorf("wait for child pid: %w", ctx.Err())
				return
			case <-ticker.C:
			}
		}
	}()

	result, err := Run(ctx, def, runOpts("main"))
	if err == nil {
		t.Fatal("expected cancellation")
	}
	if readyErr := <-pidReady; readyErr != nil {
		t.Fatal(readyErr)
	}
	if result.Steps[0].FailureReason != "canceled" {
		t.Fatalf("step = %+v", result.Steps[0])
	}
	data, readErr := os.ReadFile(pidPath)
	if readErr != nil {
		t.Fatalf("read child pid: %v", readErr)
	}
	pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
	if parseErr != nil {
		t.Fatalf("parse child pid %q: %v", data, parseErr)
	}

	deadline := time.Now().Add(time.Second)
	for {
		killErr := syscall.Kill(pid, 0)
		if errors.Is(killErr, syscall.ESRCH) {
			break
		}
		if killErr != nil {
			t.Fatalf("probe child process %d: %v", pid, killErr)
		}
		if time.Now().After(deadline) {
			t.Fatalf("child process %d survived timeout", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
