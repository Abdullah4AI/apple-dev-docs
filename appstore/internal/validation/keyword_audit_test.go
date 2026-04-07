package validation

import "testing"

func TestAuditKeywordsReportsLocaleAndCrossLocaleFindings(t *testing.T) {
	report := AuditKeywords(KeywordAuditInput{
		AppID:         "app-1",
		VersionID:     "ver-1",
		VersionString: "1.2.3",
		Platform:      "IOS",
		BlockedTerms:  []string{"free"},
		VersionLocalizations: []VersionLocalization{
			{
				ID:       "ver-loc-en",
				Locale:   "en-US",
				Keywords: "Habit Tracker,mood journal,free trial,free trial,,",
			},
			{
				ID:       "ver-loc-fr",
				Locale:   "fr-FR",
				Keywords: "habit-tracker,journal humeur",
			},
		},
		AppInfoLocalizations: []AppInfoLocalization{
			{
				ID:       "info-loc-en",
				Locale:   "en-US",
				Name:     "Habit Tracker",
				Subtitle: "Daily Mood Journal",
			},
			{
				ID:       "info-loc-fr",
				Locale:   "fr-FR",
				Name:     "Journal Humeur",
				Subtitle: "Suivi quotidien",
			},
		},
	}, false)

	if report.Summary.Errors != 0 {
		t.Fatalf("expected no errors, got %+v", report.Summary)
	}
	if report.Summary.Warnings == 0 {
		t.Fatalf("expected warnings, got %+v", report.Summary)
	}
	if report.Summary.Infos == 0 {
		t.Fatalf("expected infos, got %+v", report.Summary)
	}
	if report.Summary.Blocking != 0 {
		t.Fatalf("expected no blocking issues without strict mode, got %+v", report.Summary)
	}

	if !hasKeywordAuditCheckID(report.Checks, "metadata.keywords.locale_duplicates") {
		t.Fatalf("expected locale duplicate finding, got %+v", report.Checks)
	}
	if !hasKeywordAuditCheckID(report.Checks, "metadata.keywords.empty_segments") {
		t.Fatalf("expected empty segment finding, got %+v", report.Checks)
	}
	if !hasKeywordAuditCheckID(report.Checks, "metadata.keywords.overlap_name") {
		t.Fatalf("expected name overlap finding, got %+v", report.Checks)
	}
	if !hasKeywordAuditCheckID(report.Checks, "metadata.keywords.overlap_subtitle") {
		t.Fatalf("expected subtitle overlap finding, got %+v", report.Checks)
	}
	if !hasKeywordAuditCheckID(report.Checks, "metadata.keywords.blocked_term") {
		t.Fatalf("expected blocked term finding, got %+v", report.Checks)
	}
	if !hasKeywordAuditCheckID(report.Checks, "metadata.keywords.cross_locale_duplicates") {
		t.Fatalf("expected cross-locale duplicate finding, got %+v", report.Checks)
	}
	crossLocaleCheck, ok := findKeywordAuditCheck(report.Checks, "metadata.keywords.cross_locale_duplicates")
	if !ok {
		t.Fatalf("expected cross-locale duplicate check, got %+v", report.Checks)
	}
	if crossLocaleCheck.Keyword != "Habit Tracker" {
		t.Fatalf("expected original display keyword for cross-locale check, got %+v", crossLocaleCheck)
	}

	if len(report.Locales) != 2 {
		t.Fatalf("expected 2 locale summaries, got %+v", report.Locales)
	}
	if report.Locales[0].Locale != "en-US" {
		t.Fatalf("expected first locale en-US, got %+v", report.Locales[0])
	}
	if report.Locales[0].Warnings == 0 {
		t.Fatalf("expected en-US warnings, got %+v", report.Locales[0])
	}
	if report.Locales[0].RemainingCharacters <= 0 {
		t.Fatalf("expected remaining characters, got %+v", report.Locales[0])
	}
}

func TestAuditKeywordsStrictTurnsWarningsBlocking(t *testing.T) {
	report := AuditKeywords(KeywordAuditInput{
		AppID:        "app-1",
		VersionID:    "ver-1",
		BlockedTerms: []string{"free"},
		VersionLocalizations: []VersionLocalization{
			{
				ID:       "ver-loc-en",
				Locale:   "en-US",
				Keywords: "free trial",
			},
		},
	}, true)

	if !hasKeywordAuditCheckID(report.Checks, "metadata.keywords.blocked_term") {
		t.Fatalf("expected blocked term finding, got %+v", report.Checks)
	}
	if report.Summary.Warnings == 0 {
		t.Fatalf("expected warning count, got %+v", report.Summary)
	}
	if report.Summary.Blocking != report.Summary.Warnings {
		t.Fatalf("expected warnings to become blocking under strict mode, got %+v", report.Summary)
	}
}

func TestAuditKeywordsReportsUnderfilledBudgetAsInfo(t *testing.T) {
	report := AuditKeywords(KeywordAuditInput{
		AppID:     "app-1",
		VersionID: "ver-1",
		VersionLocalizations: []VersionLocalization{
			{
				ID:       "ver-loc-en",
				Locale:   "en-US",
				Keywords: "alpha,beta",
			},
		},
	}, false)

	if !hasKeywordAuditCheckID(report.Checks, "metadata.keywords.underfilled") {
		t.Fatalf("expected underfilled finding, got %+v", report.Checks)
	}
	if report.Summary.Infos == 0 {
		t.Fatalf("expected info count, got %+v", report.Summary)
	}
}

func hasKeywordAuditCheckID(checks []KeywordAuditCheck, id string) bool {
	_, ok := findKeywordAuditCheck(checks, id)
	return ok
}

func findKeywordAuditCheck(checks []KeywordAuditCheck, id string) (KeywordAuditCheck, bool) {
	for _, check := range checks {
		if check.ID == id {
			return check, true
		}
	}
	return KeywordAuditCheck{}, false
}
