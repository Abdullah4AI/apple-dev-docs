package validation

import (
	"strings"
	"testing"
)

func TestMetadataLengthChecks_OverLimit(t *testing.T) {
	loc := VersionLocalization{
		Locale:      "en-US",
		Description: strings.Repeat("a", LimitDescription+1),
		Keywords:    strings.Repeat("b", LimitKeywords+1),
	}
	appInfo := AppInfoLocalization{
		Locale: "en-US",
		Name:   strings.Repeat("n", LimitName+1),
	}

	checks := metadataLengthChecks([]VersionLocalization{loc}, []AppInfoLocalization{appInfo})

	if !hasCheckID(checks, "metadata.length.description") {
		t.Fatalf("expected description length check")
	}
	if !hasCheckID(checks, "metadata.length.keywords") {
		t.Fatalf("expected keywords length check")
	}
	if !hasCheckID(checks, "metadata.length.name") {
		t.Fatalf("expected name length check")
	}
}

func TestMetadataLengthChecks_Valid(t *testing.T) {
	loc := VersionLocalization{
		Locale:      "en-US",
		Description: strings.Repeat("a", LimitDescription),
		Keywords:    strings.Repeat("b", LimitKeywords),
		WhatsNew:    strings.Repeat("c", LimitWhatsNew),
	}
	appInfo := AppInfoLocalization{
		Locale:   "en-US",
		Name:     strings.Repeat("n", LimitName),
		Subtitle: strings.Repeat("s", LimitSubtitle),
	}

	checks := metadataLengthChecks([]VersionLocalization{loc}, []AppInfoLocalization{appInfo})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d", len(checks))
	}
}

func TestMetadataLengthChecks_ValidUnicode(t *testing.T) {
	loc := VersionLocalization{
		Locale:          "ja-JP",
		Description:     strings.Repeat("界", LimitDescription),
		Keywords:        strings.Repeat("語", 33),
		WhatsNew:        strings.Repeat("新", LimitWhatsNew),
		PromotionalText: strings.Repeat("宣", LimitPromotionalText),
	}
	appInfo := AppInfoLocalization{
		Locale:   "ja-JP",
		Name:     strings.Repeat("名", LimitName),
		Subtitle: strings.Repeat("副", LimitSubtitle),
	}

	checks := metadataLengthChecks([]VersionLocalization{loc}, []AppInfoLocalization{appInfo})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d", len(checks))
	}
}

func TestVersionLocalizationMinimumLengthIssues(t *testing.T) {
	issues := VersionLocalizationMinimumLengthIssues(VersionLocalization{
		Locale:      "en-US",
		Description: "TBD",
	})
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %+v", issues)
	}
	if issues[0].Field != "description" || issues[0].Length != 3 || issues[0].Minimum != MinLengthDescription {
		t.Fatalf("unexpected description issue: %+v", issues[0])
	}

	for _, description := range []string{"", strings.Repeat("d", MinLengthDescription), strings.Repeat("説", MinLengthDescription)} {
		if issues := VersionLocalizationMinimumLengthIssues(VersionLocalization{Description: description}); len(issues) != 0 {
			t.Fatalf("expected no issues for %q, got %+v", description, issues)
		}
	}
}

func TestAppInfoLocalizationMinimumLengthIssues(t *testing.T) {
	issues := AppInfoLocalizationMinimumLengthIssues(AppInfoLocalization{
		Locale: "en-US",
		Name:   "X",
	})
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %+v", issues)
	}
	if issues[0].Field != "name" || issues[0].Length != 1 || issues[0].Minimum != MinLengthName {
		t.Fatalf("unexpected name issue: %+v", issues[0])
	}

	for _, name := range []string{"", "Ab", "名前"} {
		if issues := AppInfoLocalizationMinimumLengthIssues(AppInfoLocalization{Name: name}); len(issues) != 0 {
			t.Fatalf("expected no issues for %q, got %+v", name, issues)
		}
	}
}

func longHTTPSURL(length int) string {
	const prefix = "https://example.com/"
	if length <= len(prefix) {
		return prefix[:length]
	}
	return prefix + strings.Repeat("a", length-len(prefix))
}

func TestMetadataLengthChecks_URLsOverLimit(t *testing.T) {
	loc := VersionLocalization{
		Locale:       "en-US",
		SupportURL:   longHTTPSURL(LimitSupportURL + 1),
		MarketingURL: longHTTPSURL(LimitMarketingURL + 1),
	}
	appInfo := AppInfoLocalization{
		Locale:            "en-US",
		PrivacyPolicyURL:  longHTTPSURL(LimitPrivacyPolicyURL + 1),
		PrivacyChoicesURL: longHTTPSURL(LimitPrivacyChoicesURL + 1),
	}

	checks := metadataLengthChecks([]VersionLocalization{loc}, []AppInfoLocalization{appInfo})

	wantIDs := []string{
		"metadata.length.support_url",
		"metadata.length.marketing_url",
		"metadata.length.privacy_policy_url",
		"metadata.length.privacy_choices_url",
	}
	for _, id := range wantIDs {
		if !hasCheckID(checks, id) {
			t.Fatalf("expected %s check, got %+v", id, checks)
		}
	}
	for _, check := range checks {
		if check.Severity != SeverityWarning {
			t.Fatalf("expected URL length checks to be warnings, got %+v", check)
		}
	}
}

func TestMetadataLengthChecks_URLsAtLimit(t *testing.T) {
	loc := VersionLocalization{
		Locale:       "en-US",
		SupportURL:   longHTTPSURL(LimitSupportURL),
		MarketingURL: longHTTPSURL(LimitMarketingURL),
	}
	appInfo := AppInfoLocalization{
		Locale:            "en-US",
		PrivacyPolicyURL:  longHTTPSURL(LimitPrivacyPolicyURL),
		PrivacyChoicesURL: longHTTPSURL(LimitPrivacyChoicesURL),
	}

	checks := metadataLengthChecks([]VersionLocalization{loc}, []AppInfoLocalization{appInfo})
	if len(checks) != 0 {
		t.Fatalf("expected no checks at URL limits, got %+v", checks)
	}
}

func TestVersionLocalizationLengthIssues_KeywordsUseCharacterLimit(t *testing.T) {
	keywords := strings.Repeat("語", 101)

	issues := VersionLocalizationLengthIssues(VersionLocalization{
		Locale:   "ja-JP",
		Keywords: keywords,
	})

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %+v", issues)
	}
	if issues[0].Field != "keywords" {
		t.Fatalf("expected keywords issue, got %+v", issues[0])
	}
	if issues[0].Length != 101 {
		t.Fatalf("expected keyword length 101, got %d", issues[0].Length)
	}
	if issues[0].Limit != LimitKeywords {
		t.Fatalf("expected keyword limit %d, got %d", LimitKeywords, issues[0].Limit)
	}
}
