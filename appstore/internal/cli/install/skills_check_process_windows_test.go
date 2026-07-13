//go:build windows

package install

import (
	"os/exec"
	"testing"
)

func TestConfigureDetachedSkillsCheckProcessUsesDetachedProcessGroup(t *testing.T) {
	cmd := exec.Command("unused")
	configureDetachedSkillsCheckProcess(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("Windows worker must configure process attributes")
	}
	wantFlags := uint32(detachedProcessCreationFlag | newProcessGroupCreationFlag)
	if got := cmd.SysProcAttr.CreationFlags & wantFlags; got != wantFlags {
		t.Fatalf("creation flags = %#x, want %#x", got, wantFlags)
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("Windows worker must not create a visible console window")
	}
}
