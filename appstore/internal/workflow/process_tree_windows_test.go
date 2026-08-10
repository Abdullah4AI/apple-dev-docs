//go:build windows

package workflow

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

type blockingTaskkillProcess struct {
	release <-chan struct{}
}

func (p *blockingTaskkillProcess) Start() error {
	return nil
}

func (p *blockingTaskkillProcess) Wait() error {
	<-p.release
	return nil
}

func TestConfigureProcessTreeWindows(t *testing.T) {
	command := exec.Command("cmd", "/c", "exit 0")
	configureProcessTree(command)
	if command.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if command.SysProcAttr.CreationFlags&syscall.CREATE_NEW_PROCESS_GROUP == 0 {
		t.Fatalf("CreationFlags = %#x, want CREATE_NEW_PROCESS_GROUP", command.SysProcAttr.CreationFlags)
	}
	if !command.SysProcAttr.HideWindow {
		t.Fatal("HideWindow = false, want true")
	}
	if command.Cancel == nil {
		t.Fatal("Cancel is nil")
	}
}

func TestConfigureProcessTreeWindowsCancelReturnsPromptly(t *testing.T) {
	originalNewTaskkillProcessFn := newTaskkillProcessFn
	t.Cleanup(func() {
		newTaskkillProcessFn = originalNewTaskkillProcessFn
	})

	releaseTaskkill := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(releaseTaskkill) })
	}
	t.Cleanup(release)
	newTaskkillProcessFn = func(_ context.Context, _ int) taskkillProcess {
		return &blockingTaskkillProcess{release: releaseTaskkill}
	}

	command := exec.Command(os.Args[0], "-test.run=TestProcessTreeWindowsBlockingHelper")
	command.Env = append(os.Environ(), "ASC_TEST_BLOCK_PROCESS=1")
	configureProcessTree(command)
	if err := command.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	})

	cancelResult := make(chan error, 1)
	go func() {
		cancelResult <- command.Cancel()
	}()
	select {
	case err := <-cancelResult:
		if err != nil {
			t.Fatalf("Cancel: %v", err)
		}
	case <-time.After(time.Second):
		release()
		<-cancelResult
		t.Fatal("Cancel blocked on taskkill Wait")
	}
	release()
}

func TestProcessTreeWindowsBlockingHelper(t *testing.T) {
	if os.Getenv("ASC_TEST_BLOCK_PROCESS") == "" {
		return
	}
	time.Sleep(30 * time.Second)
}
