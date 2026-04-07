package shared

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestMissingSubmitRequiredLocalizationFields_BaseFields(t *testing.T) {
	attrs := asc.AppStoreVersionLocalizationAttributes{
		Locale:      "en-US",
		Description: "A great app",
		Keywords:    "quran,islam",
		SupportURL:  "https://example.com",
	}
	missing := MissingSubmitRequiredLocalizationFields(attrs)
	if len(missing) != 0 {
		t.Fatalf("expected no missing fields, got %v", missing)
	}
}

func TestMissingSubmitRequiredLocalizationFields_AllEmpty(t *testing.T) {
	attrs := asc.AppStoreVersionLocalizationAttributes{Locale: "en-US"}
	missing := MissingSubmitRequiredLocalizationFields(attrs)
	want := []string{"description", "keywords", "supportUrl"}
	if len(missing) != len(want) {
		t.Fatalf("expected %v, got %v", want, missing)
	}
	for i, field := range want {
		if missing[i] != field {
			t.Fatalf("expected field %q at index %d, got %q", field, i, missing[i])
		}
	}
}

func TestMissingSubmitRequiredLocalizationFields_DoesNotCheckWhatsNew(t *testing.T) {
	attrs := asc.AppStoreVersionLocalizationAttributes{
		Locale:      "en-US",
		Description: "A great app",
		Keywords:    "quran,islam",
		SupportURL:  "https://example.com",
		WhatsNew:    "", // empty but should not be flagged without RequireWhatsNew
	}
	missing := MissingSubmitRequiredLocalizationFields(attrs)
	if len(missing) != 0 {
		t.Fatalf("expected no missing fields without RequireWhatsNew, got %v", missing)
	}
}

func TestMissingSubmitRequiredLocalizationFieldsWithOptions_WhatsNewRequired(t *testing.T) {
	attrs := asc.AppStoreVersionLocalizationAttributes{
		Locale:      "en-US",
		Description: "A great app",
		Keywords:    "quran,islam",
		SupportURL:  "https://example.com",
		WhatsNew:    "",
	}
	opts := SubmitReadinessOptions{RequireWhatsNew: true}
	missing := MissingSubmitRequiredLocalizationFieldsWithOptions(attrs, opts)
	if len(missing) != 1 || missing[0] != "whatsNew" {
		t.Fatalf("expected [whatsNew], got %v", missing)
	}
}

func TestMissingSubmitRequiredLocalizationFieldsWithOptions_WhatsNewPresent(t *testing.T) {
	attrs := asc.AppStoreVersionLocalizationAttributes{
		Locale:      "en-US",
		Description: "A great app",
		Keywords:    "quran,islam",
		SupportURL:  "https://example.com",
		WhatsNew:    "Bug fixes and improvements",
	}
	opts := SubmitReadinessOptions{RequireWhatsNew: true}
	missing := MissingSubmitRequiredLocalizationFieldsWithOptions(attrs, opts)
	if len(missing) != 0 {
		t.Fatalf("expected no missing fields, got %v", missing)
	}
}

func TestSubmitReadinessIssuesByLocaleWithOptions_WhatsNewMixedLocales(t *testing.T) {
	localizations := []asc.Resource[asc.AppStoreVersionLocalizationAttributes]{
		{
			ID: "loc-1",
			Attributes: asc.AppStoreVersionLocalizationAttributes{
				Locale:      "en-US",
				Description: "English description",
				Keywords:    "app,test",
				SupportURL:  "https://example.com",
				WhatsNew:    "Bug fixes",
			},
		},
		{
			ID: "loc-2",
			Attributes: asc.AppStoreVersionLocalizationAttributes{
				Locale:      "ar-SA",
				Description: "Arabic description",
				Keywords:    "تطبيق",
				SupportURL:  "https://example.com",
				WhatsNew:    "", // missing
			},
		},
		{
			ID: "loc-3",
			Attributes: asc.AppStoreVersionLocalizationAttributes{
				Locale:      "fr-FR",
				Description: "French description",
				Keywords:    "application",
				SupportURL:  "https://example.com",
				WhatsNew:    "  ", // whitespace-only
			},
		},
	}

	opts := SubmitReadinessOptions{RequireWhatsNew: true}
	issues := SubmitReadinessIssuesByLocaleWithOptions(localizations, opts)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues (ar-SA, fr-FR), got %d: %v", len(issues), issues)
	}

	// Issues should be sorted by locale
	if issues[0].Locale != "ar-SA" {
		t.Fatalf("expected first issue locale ar-SA, got %q", issues[0].Locale)
	}
	if issues[1].Locale != "fr-FR" {
		t.Fatalf("expected second issue locale fr-FR, got %q", issues[1].Locale)
	}

	for _, issue := range issues {
		if len(issue.MissingFields) != 1 || issue.MissingFields[0] != "whatsNew" {
			t.Fatalf("expected [whatsNew] for %s, got %v", issue.Locale, issue.MissingFields)
		}
	}
}

func TestSubmitReadinessIssuesByLocale_BackwardCompatible(t *testing.T) {
	localizations := []asc.Resource[asc.AppStoreVersionLocalizationAttributes]{
		{
			ID: "loc-1",
			Attributes: asc.AppStoreVersionLocalizationAttributes{
				Locale:      "en-US",
				Description: "desc",
				Keywords:    "kw",
				SupportURL:  "https://example.com",
				WhatsNew:    "", // empty but should not be flagged by default
			},
		},
	}

	issues := SubmitReadinessIssuesByLocale(localizations)
	if len(issues) != 0 {
		t.Fatalf("expected no issues from backward-compatible call, got %v", issues)
	}
}

func TestSubmitReadinessCreateWarningForLocale_ReturnsWarningForIncompleteCreate(t *testing.T) {
	attrs := asc.AppStoreVersionLocalizationAttributes{
		Locale:   "fr-FR",
		Keywords: "journal,humeur",
	}

	warning, ok := SubmitReadinessCreateWarningForLocale("", attrs, SubmitReadinessCreateModeApplied)
	if !ok {
		t.Fatal("expected warning")
	}
	if warning.Locale != "fr-FR" {
		t.Fatalf("expected locale fr-FR, got %q", warning.Locale)
	}
	if warning.Mode != SubmitReadinessCreateModeApplied {
		t.Fatalf("expected applied mode, got %q", warning.Mode)
	}
	want := []string{"description", "supportUrl"}
	if len(warning.MissingFields) != len(want) {
		t.Fatalf("expected %v, got %v", want, warning.MissingFields)
	}
	for i := range want {
		if warning.MissingFields[i] != want[i] {
			t.Fatalf("expected missing field %q at index %d, got %q", want[i], i, warning.MissingFields[i])
		}
	}
}

func TestSubmitReadinessCreateWarningForLocale_ReturnsFalseForCompleteCreate(t *testing.T) {
	attrs := asc.AppStoreVersionLocalizationAttributes{
		Locale:      "en-US",
		Description: "Description",
		Keywords:    "app,test",
		SupportURL:  "https://example.com/support",
	}

	if _, ok := SubmitReadinessCreateWarningForLocale("", attrs, SubmitReadinessCreateModePlanned); ok {
		t.Fatal("expected no warning")
	}
}

func TestSubmitReadinessCreateWarningForLocaleWithOptions_RequiresWhatsNew(t *testing.T) {
	attrs := asc.AppStoreVersionLocalizationAttributes{
		Locale:      "en-US",
		Description: "Description",
		Keywords:    "app,test",
		SupportURL:  "https://example.com/support",
	}

	warning, ok := SubmitReadinessCreateWarningForLocaleWithOptions(
		"",
		attrs,
		SubmitReadinessCreateModeApplied,
		SubmitReadinessOptions{RequireWhatsNew: true},
	)
	if !ok {
		t.Fatal("expected warning")
	}
	if warning.Locale != "en-US" {
		t.Fatalf("expected locale en-US, got %q", warning.Locale)
	}
	if warning.Mode != SubmitReadinessCreateModeApplied {
		t.Fatalf("expected applied mode, got %q", warning.Mode)
	}
	if len(warning.MissingFields) != 1 || warning.MissingFields[0] != "whatsNew" {
		t.Fatalf("expected missing fields [whatsNew], got %+v", warning.MissingFields)
	}
}

func TestNormalizeSubmitReadinessCreateWarnings_SortsAndDedupes(t *testing.T) {
	warnings := []SubmitReadinessCreateWarning{
		{
			Locale:        "fr-FR",
			Mode:          SubmitReadinessCreateModeApplied,
			MissingFields: []string{"supportUrl"},
		},
		{
			Locale:        "EN-us",
			Mode:          SubmitReadinessCreateModePlanned,
			MissingFields: []string{"keywords"},
		},
		{
			Locale:        "en-US",
			Mode:          SubmitReadinessCreateModePlanned,
			MissingFields: []string{"description", "keywords"},
		},
		{
			Locale:        "fr-FR",
			Mode:          SubmitReadinessCreateModeApplied,
			MissingFields: []string{"description"},
		},
	}

	normalized := NormalizeSubmitReadinessCreateWarnings(warnings)
	if len(normalized) != 2 {
		t.Fatalf("expected 2 warnings, got %d: %+v", len(normalized), normalized)
	}
	if normalized[0].Locale != "EN-us" || normalized[0].Mode != SubmitReadinessCreateModePlanned {
		t.Fatalf("expected planned EN-us warning first, got %+v", normalized[0])
	}
	if normalized[1].Locale != "fr-FR" || normalized[1].Mode != SubmitReadinessCreateModeApplied {
		t.Fatalf("expected applied fr-FR warning second, got %+v", normalized[1])
	}
	wantSecond := []string{"description", "supportUrl"}
	if len(normalized[1].MissingFields) != len(wantSecond) {
		t.Fatalf("expected merged fields %v, got %v", wantSecond, normalized[1].MissingFields)
	}
	for i := range wantSecond {
		if normalized[1].MissingFields[i] != wantSecond[i] {
			t.Fatalf("expected missing field %q at index %d, got %q", wantSecond[i], i, normalized[1].MissingFields[i])
		}
	}
}

func TestFormatSubmitReadinessCreateWarning_DistinguishesMode(t *testing.T) {
	planned := FormatSubmitReadinessCreateWarning(SubmitReadinessCreateWarning{
		Locale:        "de-DE",
		Mode:          SubmitReadinessCreateModePlanned,
		MissingFields: []string{"description", "supportUrl"},
	})
	if !strings.Contains(planned, "creating locale de-DE would make it participate in submission validation") {
		t.Fatalf("expected planned wording, got %q", planned)
	}

	applied := FormatSubmitReadinessCreateWarning(SubmitReadinessCreateWarning{
		Locale:        "de-DE",
		Mode:          SubmitReadinessCreateModeApplied,
		MissingFields: []string{"description", "supportUrl"},
	})
	if !strings.Contains(applied, "created locale de-DE now participates in submission validation") {
		t.Fatalf("expected applied wording, got %q", applied)
	}
}

func TestPrintSubmitReadinessCreateWarnings_NormalizesBeforePrinting(t *testing.T) {
	var stderr bytes.Buffer
	err := PrintSubmitReadinessCreateWarnings(&stderr, []SubmitReadinessCreateWarning{
		{
			Locale:        "fr-FR",
			Mode:          SubmitReadinessCreateModeApplied,
			MissingFields: []string{"supportUrl"},
		},
		{
			Locale:        "fr-FR",
			Mode:          SubmitReadinessCreateModeApplied,
			MissingFields: []string{"description"},
		},
		{
			Locale:        "en-US",
			Mode:          SubmitReadinessCreateModePlanned,
			MissingFields: []string{"keywords"},
		},
	})
	if err != nil {
		t.Fatalf("PrintSubmitReadinessCreateWarnings() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 warning lines, got %d: %q", len(lines), stderr.String())
	}
	if !strings.Contains(lines[0], "en-US") || !strings.Contains(lines[0], "creating locale") {
		t.Fatalf("expected planned en-US warning first, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "fr-FR") || !strings.Contains(lines[1], "description, supportUrl") {
		t.Fatalf("expected merged fr-FR warning second, got %q", lines[1])
	}
}
