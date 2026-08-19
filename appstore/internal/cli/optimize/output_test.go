package optimize

import (
	"slices"
	"strings"
	"testing"
)

func TestSearchPlanCompactTableIsScreenshotFriendly(t *testing.T) {
	report := SearchPlanReport{
		AppID:    "123456789",
		Version:  "1.3.1",
		Platform: "IOS",
		Country:  "US",
		Genre:    "PRODUCTIVITY_UTILITIES",
		Locale:   "en-US",
		Metadata: searchMetadataSnapshot{Name: "Zenther: AI Calorie Tracker"},
		Summary: SearchPlanSummary{
			Terms:              516,
			AvailableSources:   5,
			EmptySources:       5,
			UnavailableSources: 2,
		},
	}

	rows := searchPlanCompactSummaryRows(report)
	for _, want := range []string{
		"Zenther: AI Calorie Tracker",
		"1.3.1 · iOS",
		"US · en-US",
		"5 available · 5 empty · 2 unavailable",
	} {
		if !tableRowsContain(rows, want) {
			t.Fatalf("compact summary missing %q: %#v", want, rows)
		}
	}
	if tableRowsContain(rows, report.AppID) {
		t.Fatalf("compact summary exposes app ID: %#v", rows)
	}

	wantHeaders := []string{"Term", "Popularity", "Genre Rank", "Share", "Installs", "CPA", "Next Step", "Confidence"}
	if got := searchPlanCompactHeaders(); !slices.Equal(got, wantHeaders) {
		t.Fatalf("compact headers = %v, want %v", got, wantHeaders)
	}
	row := SearchPlanRow{
		Term:          "calorie tracker",
		Popularity5:   intPtr(4),
		Popularity100: intPtr(79),
		Actions:       []string{"metadata_candidate", "untested_candidate"},
		Confidence:    "suggested",
	}
	got := searchPlanCompactRows([]SearchPlanRow{row})
	if len(got) != 1 || !slices.Equal(got[0], []string{"calorie tracker", "4 / 79", "—", "—", "—", "—", "metadata · test", "suggested"}) {
		t.Fatalf("compact row = %#v", got)
	}
}

func TestCompactSearchPlanDiagnosticRemovesProviderHTMLAndCapsLength(t *testing.T) {
	for _, marker := range []string{
		"<html><head><title>500 Internal Server Error</title></head>",
		"<!DOCTYPE html><html><head><title>500 Internal Server Error</title></head>",
		"<body>Apple returned a provider fragment",
	} {
		errorText := "retry limit exceeded after 4 retries: HTTP 500: " + marker + "Apple returned a very long response that must not stretch the terminal table beyond a useful screenshot width"
		got := compactSearchPlanDiagnostic(errorText)
		if strings.Contains(strings.ToLower(got), "html") || strings.Contains(got, "provider fragment") || strings.Contains(got, "Internal Server Error") {
			t.Fatalf("compact diagnostic retained provider HTML from %q: %q", marker, got)
		}
		if len([]rune(got)) > 72 {
			t.Fatalf("compact diagnostic length = %d, want <= 72: %q", len([]rune(got)), got)
		}
		if got != "retry limit exceeded after 4 retries: HTTP 500" {
			t.Fatalf("compact diagnostic = %q", got)
		}
	}
}

func tableRowsContain(rows [][]string, want string) bool {
	for _, row := range rows {
		for _, value := range row {
			if value == want {
				return true
			}
		}
	}
	return false
}
