package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAppsPublishedReportUsesCamelCaseJSONFields(t *testing.T) {
	report := AppsPublishedReport{
		AuditedAppCount:   2,
		PublishedAppCount: 1,
		Apps: []PublishedApp{{
			ID:                      "app-1",
			BundleID:                "com.example.app",
			AvailabilityID:          "availability-1",
			PublishedTerritoryCount: 3,
		}},
		Failures: []PublishedAppFailure{{ID: "app-2", Error: "request failed"}},
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	for _, want := range []string{"auditedAppCount", "publishedAppCount", "bundleId", "availabilityId", "publishedTerritoryCount", "failures"} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("JSON output missing camelCase field %q: %s", want, encoded)
		}
	}
}

func TestAppsPublishedReportRegisteredTableAndMarkdownOutput(t *testing.T) {
	report := &AppsPublishedReport{
		AuditedAppCount:   2,
		PublishedAppCount: 1,
		Apps: []PublishedApp{{
			ID:                      "app-1",
			Name:                    "Healthy",
			BundleID:                "com.example.healthy",
			SKU:                     "HEALTHY",
			AvailabilityID:          "availability-1",
			PublishedTerritoryCount: 2,
		}},
		Failures: []PublishedAppFailure{{
			ID:    "app-2",
			Name:  "Broken",
			Error: "request failed",
		}},
	}

	ensureOutputRegistryPopulated()
	if !isRegistryTypeRegistered(typeForPtr[AppsPublishedReport]()) {
		t.Fatal("AppsPublishedReport is not registered with the output renderer")
	}

	table := captureStdout(t, func() error { return PrintTable(report) })
	for _, want := range []string{"Healthy", "Failed app audits:", "Broken", "request failed", "Audited 2 app records; found 1 published app. 1 app audit(s) failed."} {
		if !strings.Contains(table, want) {
			t.Fatalf("table output missing %q: %s", want, table)
		}
	}

	markdown := captureStdout(t, func() error { return PrintMarkdown(report) })
	for _, want := range []string{"| Healthy", "Failed app audits:", "| Broken", "| request failed", "Audited 2 app records; found 1 published app. 1 app audit(s) failed."} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("Markdown output missing %q: %s", want, markdown)
		}
	}
}
