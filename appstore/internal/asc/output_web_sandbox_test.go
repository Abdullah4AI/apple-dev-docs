package asc

import (
	"strings"
	"testing"
)

func TestPrintWebSandboxDeleteResultUsesRegistry(t *testing.T) {
	result := &WebSandboxDeleteResult{IDs: []string{"tester-one", "tester-two"}, Deleted: true}

	outputs := map[string]string{
		"table":    captureStdout(t, func() error { return PrintTable(result) }),
		"markdown": captureStdout(t, func() error { return PrintMarkdown(result) }),
	}
	for name, output := range outputs {
		for _, want := range []string{"ID", "Deleted", "tester-one", "tester-two", "true"} {
			if !strings.Contains(output, want) {
				t.Fatalf("%s output missing %q: %q", name, want, output)
			}
		}
	}
}
