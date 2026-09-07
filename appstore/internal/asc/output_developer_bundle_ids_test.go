package asc

import (
	"strings"
	"testing"
)

func TestDeveloperBundleIDCapabilityDisableResultUsesRegisteredRenderers(t *testing.T) {
	result := &DeveloperBundleIDCapabilityDisableResult{
		BundleID:   "bundle-1",
		Capability: "PRIVATE_CLOUD_COMPUTE",
		Enabled:    false,
		Changed:    true,
		Status:     "disabled",
	}

	ensureOutputRegistryPopulated()
	if !isRegistryTypeRegistered(typeForPtr[DeveloperBundleIDCapabilityDisableResult]()) {
		t.Fatal("DeveloperBundleIDCapabilityDisableResult is not registered with the output renderer")
	}

	table := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"Bundle ID", "Capability", "Enabled", "Changed", "Status", "bundle-1", "PRIVATE_CLOUD_COMPUTE", "false", "true", "disabled"} {
		if !strings.Contains(table, want) {
			t.Fatalf("table output missing %q: %q", want, table)
		}
	}

	markdown := captureStdout(t, func() error { return PrintMarkdown(result) })
	for _, want := range []string{"Bundle ID", "Capability", "Enabled", "Changed", "Status", "bundle-1", "PRIVATE_CLOUD_COMPUTE", "false", "true", "disabled"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown output missing %q: %q", want, markdown)
		}
	}
}
