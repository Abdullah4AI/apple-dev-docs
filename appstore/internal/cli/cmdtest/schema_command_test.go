package cmdtest

import (
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

func TestRun_SchemaInvalidMethodReturnsUsage(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")

	_, stderr := captureOutput(t, func() {
		code := cmd.Run([]string{"schema", "--list", "--method", "DELTE"}, "1.0.0")
		if code != cmd.ExitUsage {
			t.Fatalf("expected exit code %d, got %d", cmd.ExitUsage, code)
		}
	})

	if !strings.Contains(stderr, "invalid --method") {
		t.Fatalf("expected invalid --method error, got stderr: %s", stderr)
	}
}

func TestRun_SchemaValidMethodReturnsSuccess(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")

	stdout, stderr := captureOutput(t, func() {
		code := cmd.Run([]string{"schema", "--list", "--method", "DELETE"}, "1.0.0")
		if code != cmd.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d", cmd.ExitSuccess, code)
		}
	})

	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %s", stderr)
	}
	if !strings.Contains(stdout, "\"method\":\"DELETE\"") {
		t.Fatalf("expected DELETE endpoints in output, got: %s", stdout)
	}
}
