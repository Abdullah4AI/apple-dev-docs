//go:build darwin || linux

package telemetry

import (
	"os/exec"
	"testing"
)

func TestConfigureDetachedProcessCreatesNewSession(t *testing.T) {
	command := &exec.Cmd{}
	configureDetachedProcess(command)
	if command.SysProcAttr == nil || !command.SysProcAttr.Setsid {
		t.Fatalf("detached SysProcAttr = %+v, want Setsid", command.SysProcAttr)
	}
}
