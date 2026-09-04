package asc

import (
	"strings"
	"testing"
)

func TestPrintTableWebAppDeleteResultUsesRegistry(t *testing.T) {
	result := &WebAppDeleteResult{
		AppID:    "1234567890",
		Name:     "Throwaway",
		BundleID: "com.example.throwaway",
		Removed:  true,
	}

	output := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"1234567890", "Throwaway", "com.example.throwaway", "true"} {
		if !strings.Contains(output, want) {
			t.Fatalf("table output missing %q: %q", want, output)
		}
	}
}

func TestPrintMarkdownWebAppDeleteResultIncludesDryRun(t *testing.T) {
	result := &WebAppDeleteResult{
		AppID:    "1234567890",
		Name:     "Throwaway",
		BundleID: "com.example.throwaway",
		Removed:  false,
		DryRun:   true,
	}

	output := captureStdout(t, func() error { return PrintMarkdown(result) })
	for _, want := range []string{"1234567890", "false", "true"} {
		if !strings.Contains(output, want) {
			t.Fatalf("markdown output missing %q: %q", want, output)
		}
	}
}
