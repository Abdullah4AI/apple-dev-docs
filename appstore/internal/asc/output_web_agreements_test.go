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
