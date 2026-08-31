package validation

import (
	"strings"
	"testing"
)

func TestMetadataQualityChecksMapsDeterministicWarningsWithoutKeywordValues(t *testing.T) {
	checks := MetadataQualityChecks(
		[]VersionLocalization{{
			ID:       "version-loc-1",
			Locale:   "en-US",
			Keywords: "alpha,alpha,,beta",
		}},
		[]AppInfoLocalization{{
			ID:       "app-info-loc-1",
			Locale:   "en-US",
			Name:     "X",
			Subtitle: "Beta",
		}, {
			ID:       "app-info-loc-2",
			Locale:   "en-US",
			Name:     "Alpha",
			Subtitle: "Beta",
		}},
	)

	wantIDs := []string{
		"metadata.minimum.name",
		"metadata.keywords.empty_segments",
		"metadata.keywords.locale_duplicates",
		"metadata.keywords.overlap_name",
		"metadata.keywords.overlap_subtitle",
	}
	if got := checkIDs(checks); !equalStrings(got, wantIDs) {
		t.Fatalf("check IDs = %v, want %v", got, wantIDs)
	}
	for _, check := range checks {
		if check.Severity != SeverityWarning {
			t.Fatalf("check = %+v, want warning", check)
		}
		if check.ResourceType == "" || check.ResourceID == "" || check.Locale != "en-US" {
			t.Fatalf("check = %+v, want localization context", check)
		}
		if strings.Contains(strings.ToLower(check.Message), "alpha") || strings.Contains(strings.ToLower(check.Message), "beta") {
			t.Fatalf("check leaked keyword value: %+v", check)
		}
	}
}

func TestMetadataQualityChecksSkipsInformationalAndUserSuppliedKeywordRules(t *testing.T) {
	checks := MetadataQualityChecks(
		[]VersionLocalization{{
			ID:       "version-loc-1",
			Locale:   "en-US",
			Keywords: "alpha,beta",
		}},
		[]AppInfoLocalization{{
			ID:       "app-info-loc-1",
			Locale:   "en-US",
			Name:     "A useful app",
			Subtitle: "Track routines",
		}},
	)
	if len(checks) != 0 {
		t.Fatalf("checks = %+v, want no warning-only quality findings", checks)
	}
}

func TestMetadataQualityChecksCountsAppNameRunes(t *testing.T) {
	for _, test := range []struct {
		name      string
		appName   string
		wantIssue bool
	}{
		{name: "one multibyte rune", appName: "界", wantIssue: true},
		{name: "two multibyte runes", appName: "界面", wantIssue: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			checks := MetadataQualityChecks(nil, []AppInfoLocalization{{
				ID:     "app-info-loc-1",
				Locale: "zh-Hans",
				Name:   test.appName,
			}})
			gotIssue := hasCheckID(checks, "metadata.minimum.name")
			if gotIssue != test.wantIssue {
				t.Fatalf("checks = %+v, want metadata.minimum.name=%t", checks, test.wantIssue)
			}
		})
	}
}

func TestValidateIncludesMetadataQualityWarningsAndStrictModeControlsBlocking(t *testing.T) {
	report := Validate(Input{
		VersionLocalizations: []VersionLocalization{{
			ID:       "version-loc-1",
			Locale:   "en-US",
			Keywords: "alpha,alpha",
		}},
		AppInfoLocalizations: []AppInfoLocalization{{
			ID:     "app-info-loc-1",
			Locale: "en-US",
			Name:   "X",
		}},
	}, false)
	if !hasCheckID(report.Checks, "metadata.minimum.name") || !hasCheckID(report.Checks, "metadata.keywords.locale_duplicates") {
		t.Fatalf("checks = %+v, want metadata quality warnings", report.Checks)
	}
	if report.Summary.Blocking != report.Summary.Errors {
		t.Fatalf("summary = %+v, want warnings non-blocking without strict mode", report.Summary)
	}

	strictReport := Validate(Input{
		VersionLocalizations: []VersionLocalization{{
			ID:       "version-loc-1",
			Locale:   "en-US",
			Keywords: "alpha,alpha",
		}},
		AppInfoLocalizations: []AppInfoLocalization{{
			ID:     "app-info-loc-1",
			Locale: "en-US",
			Name:   "X",
		}},
	}, true)
	if strictReport.Summary.Blocking != strictReport.Summary.Errors+strictReport.Summary.Warnings {
		t.Fatalf("strict summary = %+v, want warnings to block", strictReport.Summary)
	}
}

func checkIDs(checks []CheckResult) []string {
	ids := make([]string, 0, len(checks))
	for _, check := range checks {
		ids = append(ids, check.ID)
	}
	return ids
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
