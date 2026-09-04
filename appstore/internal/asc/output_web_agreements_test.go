package asc

import (
	"strings"
	"testing"
)

func TestPrintTableWebAgreementsStatusIncludesBannerOnlyPendingState(t *testing.T) {
	result := &WebAgreementsStatusResult{
		TeamID:  "TEAM123456",
		Pending: true,
		ContractMessages: []WebAgreementContractMessage{{
			ID:      "contract_message",
			Group:   "Alert",
			Subject: "Apple Developer Program License Agreement Updated",
			Message: "Review the updated agreement.",
		}},
	}

	output := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"TEAM123456", "true", "Apple Developer Program License Agreement Updated"} {
		if !strings.Contains(output, want) {
			t.Fatalf("table output missing %q: %q", want, output)
		}
	}
}

func TestPrintMarkdownWebAgreementsAcceptResultUsesRegistry(t *testing.T) {
	result := &WebAgreementsAcceptResult{
		TeamID:       "TEAM123456",
		AgreementIDs: []string{"XG8DNV4HYY"},
		Status:       "accepted",
		Agreements: []WebAgreement{{
			AgreementID:  "XG8DNV4HYY",
			DateAccepted: "2026-08-19T16:56:47Z",
		}},
	}

	output := captureStdout(t, func() error { return PrintMarkdown(result) })
	for _, want := range []string{"TEAM123456", "XG8DNV4HYY", "accepted", "2026-08-19T16:56:47Z"} {
		if !strings.Contains(output, want) {
			t.Fatalf("markdown output missing %q: %q", want, output)
		}
	}
}

func TestPrintTableWebAgreementsAcceptResultRendersOneRowPerAgreement(t *testing.T) {
	result := &WebAgreementsAcceptResult{
		TeamID:       "TEAM123456",
		AgreementIDs: []string{"XG8DNV4HYY", "AB12CD34EF"},
		Status:       "accepted",
		Verified:     true,
		Agreements: []WebAgreement{
			{AgreementID: "XG8DNV4HYY", Status: "active", DateAccepted: "2026-08-19T16:56:47Z"},
			{AgreementID: "AB12CD34EF", Status: "active", DateAccepted: "2026-09-01T08:00:00Z"},
		},
	}

	output := captureStdout(t, func() error { return PrintTable(result) })
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var first, second string
	for _, line := range lines {
		switch {
		case strings.Contains(line, "XG8DNV4HYY"):
			first = line
		case strings.Contains(line, "AB12CD34EF"):
			second = line
		}
	}
	if first == "" || second == "" || first == second {
		t.Fatalf("table output must render one row per agreement: %q", output)
	}
	if !strings.Contains(first, "2026-08-19T16:56:47Z") || strings.Contains(first, "2026-09-01T08:00:00Z") {
		t.Fatalf("row for XG8DNV4HYY must carry only its own acceptance time: %q", first)
	}
	if !strings.Contains(second, "2026-09-01T08:00:00Z") || strings.Contains(second, "2026-08-19T16:56:47Z") {
		t.Fatalf("row for AB12CD34EF must carry only its own acceptance time: %q", second)
	}
	for _, want := range []string{"TEAM123456", "accepted", "true"} {
		if !strings.Contains(first, want) {
			t.Fatalf("row %q missing %q", first, want)
		}
	}
}

func TestPrintTableWebAgreementDownloadResultUsesRegistry(t *testing.T) {
	result := &WebAgreementDownloadResult{
		AgreementID:  "XG8DNV4HYY",
		TeamID:       "TEAM123456",
		Title:        "Apple Developer Program License Agreement",
		Version:      "5031",
		Path:         "./agreement.pdf",
		BytesWritten: 1234,
		ContentType:  "application/pdf",
	}

	output := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"Agreement ID", "Path", "Bytes", "Content Type", "XG8DNV4HYY", "TEAM123456", "./agreement.pdf", "1234", "application/pdf"} {
		if !strings.Contains(output, want) {
			t.Fatalf("table output missing %q: %q", want, output)
		}
	}
}
