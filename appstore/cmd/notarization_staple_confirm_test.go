package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRunNotarizationStapleRequiresConfirmationBeforeTargetValidation(t *testing.T) {
	resetReportFlags(t)

	missingTarget := filepath.Join(t.TempDir(), "missing.dmg")
	stdout, stderr := captureCommandOutput(t, func() {
		if code := Run([]string{"notarization", "staple", "--file", missingTarget}, "4.12.0"); code != ExitUsage {
			t.Fatalf("Run() exit code = %d, want %d", code, ExitUsage)
		}
	})
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "--confirm is required") {
		t.Fatalf("stderr = %q, want missing-confirm diagnostic", stderr)
	}
	if strings.Contains(stderr, "does not exist") {
		t.Fatalf("stderr = %q, target validation ran before confirmation", stderr)
	}
}
