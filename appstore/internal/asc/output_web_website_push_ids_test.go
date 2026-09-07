package asc

import (
	"strings"
	"testing"
)

func TestPrintTableWebWebsitePushIDMutationReceiptUsesRegistry(t *testing.T) {
	result := &WebWebsitePushIDMutationResult{
		Operation:     "create",
		WebsitePushID: "5D2243QPXH",
		Identifier:    "web.example.com",
		Name:          "Example Website",
		Changed:       true,
		Verified:      true,
		Status:        "created",
	}
	for _, render := range []func(any) string{
		func(value any) string { return captureStdout(t, func() error { return PrintTable(value) }) },
		func(value any) string { return captureStdout(t, func() error { return PrintMarkdown(value) }) },
	} {
		output := render(result)
		for _, want := range []string{"Operation", "Website Push ID", "Identifier", "Example Website", "5D2243QPXH", "true", "created"} {
			if !strings.Contains(output, want) {
				t.Fatalf("output %q missing %q", output, want)
			}
		}
	}
}
