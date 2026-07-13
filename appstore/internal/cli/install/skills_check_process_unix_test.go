//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package install

import (
	"os/exec"
	"testing"
)

func TestConfigureDetachedSkillsCheckProcessUsesNewSession(t *testing.T) {
	cmd := exec.Command("unused")
	configureDetachedSkillsCheckProcess(cmd)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setsid {
		t.Fatal("Unix worker must start in a detached session")
	}
}
