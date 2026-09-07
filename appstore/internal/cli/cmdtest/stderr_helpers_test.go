package cmdtest

import (
	"strings"
	"testing"
)

// assertUsageDiagnosticFirstLine locks a usage error's diagnostic as the
// complete first line of stderr, so a leading warning, trailing usage text or
// any extra wording in the diagnostic itself fails the assertion.
//
// wantMessage values ending in ":" carry a variable tail the command does not
// own - the URL parser's own detail behind "--next must be a valid URL:" - and
// are matched as a prefix. Every other diagnostic is compared exactly.
func assertUsageDiagnosticFirstLine(t *testing.T, stderr, wantMessage string) {
	t.Helper()

	diagnostic, _, _ := strings.Cut(stderr, "\n")
	want := "Error: " + wantMessage
	if strings.HasSuffix(wantMessage, ":") {
		if !strings.HasPrefix(diagnostic, want) {
			t.Fatalf("stderr first line = %q, want prefix %q", diagnostic, want)
		}
		return
	}
	if diagnostic != want {
		t.Fatalf("stderr first line = %q, want %q", diagnostic, want)
	}
}

func assertEmptyStderr(t *testing.T, stderr string) {
	t.Helper()

	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func requireStderrContainsWarning(t *testing.T, stderr, warning string) {
	t.Helper()
	if !strings.Contains(stderr, warning) {
		t.Fatalf("expected stderr to contain warning %q, got %q", warning, stderr)
	}
}
