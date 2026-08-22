package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDeveloperSystemStatusReportJSONContract(t *testing.T) {
	epochEnd := int64(1787086740000)
	report := &DeveloperSystemStatusReport{
		Source:  "https://developer.apple.com/system-status/",
		Message: "Developer services are operating in disaster recovery mode.",
		Summary: DeveloperSystemStatusSummary{
			Status:              "issues",
			TotalServices:       2,
			OperationalServices: 1,
			AffectedServices:    1,
			ActiveIncidents:     1,
		},
		Services: []DeveloperSystemStatusService{{
			Name:   "App Store Connect API",
			Status: "issues",
			Events: []DeveloperSystemStatusEvent{{
				MessageID:    "event-1",
				EventStatus:  "ongoing",
				EpochEndDate: &epochEnd,
				Active:       true,
			}},
		}},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	output := string(data)
	for _, field := range []string{`"message":"Developer services are operating in disaster recovery mode."`, `"totalServices":2`, `"operationalServices":1`, `"affectedServices":1`, `"activeIncidents":1`, `"messageId":"event-1"`, `"epochEndDate":1787086740000`, `"active":true`} {
		if !strings.Contains(output, field) {
			t.Fatalf("JSON output missing %s: %s", field, output)
		}
	}
}

func TestDeveloperSystemStatusReportRendersTableAndMarkdown(t *testing.T) {
	report := &DeveloperSystemStatusReport{
		Source:  "https://developer.apple.com/system-status/",
		Message: "Developer services are operating in disaster recovery mode.",
		Summary: DeveloperSystemStatusSummary{
			Status:           "issues",
			TotalServices:    1,
			AffectedServices: 1,
			ActiveIncidents:  1,
		},
		Services: []DeveloperSystemStatusService{{
			Name:   "Xcode Cloud",
			Status: "issues",
			Events: []DeveloperSystemStatusEvent{{
				StatusType:  "Outage",
				Message:     "Users are experiencing a problem.",
				DatePosted:  "08/18/2026 13:01 PDT",
				EventStatus: "ongoing",
				Active:      true,
			}},
		}},
	}

	for _, renderer := range []struct {
		name string
		fn   func(any) error
	}{
		{name: "table", fn: PrintTable},
		{name: "markdown", fn: PrintMarkdown},
	} {
		t.Run(renderer.name, func(t *testing.T) {
			assertRenderedNonJSONContains(t, renderer.fn, report,
				"Active Incidents", "Developer services are operating in disaster recovery mode.",
				"Xcode Cloud", "Outage", "Users are experiencing a problem.")
		})
	}
}
