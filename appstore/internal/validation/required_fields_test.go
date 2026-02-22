package validation

import "testing"

func TestRequiredFieldChecks_MissingPrimaryLocale(t *testing.T) {
	checks := requiredFieldChecks("en-US", "1.2.3", []VersionLocalization{
		{Locale: "fr-FR", Description: "desc", Keywords: "kw", SupportURL: "https://example.com"},
	}, []AppInfoLocalization{
		{Locale: "fr-FR", Name: "Name"},
	})

	if !hasCheckID(checks, "metadata.required.primary_locale") {
		t.Fatalf("expected primary locale check")
	}
}

func TestRequiredFieldChecks_MissingFields(t *testing.T) {
	checks := requiredFieldChecks("", "1.2.3", []VersionLocalization{
		{Locale: "en-US"},
	}, []AppInfoLocalization{
		{Locale: "en-US"},
	})

	if !hasCheckID(checks, "metadata.required.description") {
		t.Fatalf("expected description required check")
	}
	if !hasCheckID(checks, "metadata.required.keywords") {
		t.Fatalf("expected keywords required check")
	}
	if !hasCheckID(checks, "metadata.required.support_url") {
		t.Fatalf("expected support url required check")
	}
	if !hasCheckID(checks, "metadata.required.name") {
		t.Fatalf("expected name required check")
	}
}

func TestRequiredFieldChecks_SkipsWhatsNewOnInitialRelease(t *testing.T) {
	checks := requiredFieldChecks("", "1.0", []VersionLocalization{
		{Locale: "en-US", Description: "desc", Keywords: "kw", SupportURL: "https://example.com"},
	}, []AppInfoLocalization{
		{Locale: "en-US", Name: "Name"},
	})

	if hasCheckID(checks, "metadata.required.whats_new") {
		t.Fatalf("did not expect whatsNew warning for initial release")
	}
}

func TestRequiredFieldChecks_WarnsWhatsNewOnUpdateRelease(t *testing.T) {
	checks := requiredFieldChecks("", "1.0.1", []VersionLocalization{
		{Locale: "en-US", Description: "desc", Keywords: "kw", SupportURL: "https://example.com"},
	}, []AppInfoLocalization{
		{Locale: "en-US", Name: "Name"},
	})

	if !hasCheckID(checks, "metadata.required.whats_new") {
		t.Fatalf("expected whatsNew warning for update release")
	}
}
