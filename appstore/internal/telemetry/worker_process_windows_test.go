//go:build windows

package telemetry

import (
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

func TestConfigureDetachedProcessUsesWindowsDetachFlags(t *testing.T) {
	command := &exec.Cmd{}
	configureDetachedProcess(command)
	if command.SysProcAttr == nil {
		t.Fatal("detached SysProcAttr is nil")
	}
	want := uint32(windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP)
	if got := command.SysProcAttr.CreationFlags & want; got != want {
		t.Fatalf("CreationFlags = %#x, want %#x", command.SysProcAttr.CreationFlags, want)
	}
	if !command.SysProcAttr.HideWindow {
		t.Fatal("detached worker window is visible")
	}
}
