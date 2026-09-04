package cmdtest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

func TestRun_CapabilitiesReportsTaxCategoryGap(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")

	stdout, stderr := captureOutput(t, func() {
		code := cmd.Run([]string{"capabilities", "--area", "monetization", "--output", "json"}, "1.0.0")
		if code != cmd.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d", cmd.ExitSuccess, code)
		}
	})

	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %s", stderr)
	}

	var resp capabilitiesTestResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("expected JSON output, got error %v and stdout %s", err, stdout)
	}

	assertCapability(t, resp, "App and In-App Purchase tax category", "not-public-api", "")

	for _, entry := range resp.Capabilities {
		if entry.Capability != "App and In-App Purchase tax category" {
			continue
		}
		if len(entry.Commands) != 0 {
			t.Fatalf("expected no commands for the tax category gap, got %v", entry.Commands)
		}
		return
	}
}
