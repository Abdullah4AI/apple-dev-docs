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
