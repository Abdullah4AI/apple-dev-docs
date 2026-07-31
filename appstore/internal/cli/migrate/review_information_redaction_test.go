package migrate

import (
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

const migrateRedactionSentinel = "asc-red-sentinel-migrate-unit-pw-6d0b12"

func TestPresentableImportResultKeepsRealPasswordForRequests(t *testing.T) {
	password := migrateRedactionSentinel
	name := "reviewer@example.com"
	info := &ReviewInformation{DemoAccountName: &name, DemoAccountPassword: &password}
	result := &MigrateImportResult{VersionID: "version-1", ReviewInformation: info}

	safe := presentableImportResult(result, false)

	if safe.ReviewInformation.DemoAccountPassword == nil ||
		*safe.ReviewInformation.DemoAccountPassword != asc.RedactedValuePlaceholder {
		t.Fatalf("rendered password = %v, want placeholder", safe.ReviewInformation.DemoAccountPassword)
	}
	if safe.ReviewInformation.DemoAccountName == nil || *safe.ReviewInformation.DemoAccountName != name {
		t.Fatalf("redaction dropped the demo account name: %+v", safe.ReviewInformation)
	}
	if *info.DemoAccountPassword != migrateRedactionSentinel {
		t.Fatalf("redaction mutated the imported password to %q", *info.DemoAccountPassword)
	}

	attrs := buildReviewDetailUpdateAttributes(result.ReviewInformation)
	if attrs.DemoAccountPassword == nil || *attrs.DemoAccountPassword != migrateRedactionSentinel {
		t.Fatalf("update attributes password = %v, want the real value", attrs.DemoAccountPassword)
	}
	createAttrs := buildReviewDetailCreateAttributes(result.ReviewInformation)
	if createAttrs.DemoAccountPassword == nil || *createAttrs.DemoAccountPassword != migrateRedactionSentinel {
		t.Fatalf("create attributes password = %v, want the real value", createAttrs.DemoAccountPassword)
	}
}

func TestPresentableImportResultComparesAgainstRealPassword(t *testing.T) {
	password := migrateRedactionSentinel
	info := &ReviewInformation{DemoAccountPassword: &password}
	result := &MigrateImportResult{ReviewInformation: info}

	_ = presentableImportResult(result, false)

	existing := asc.AppStoreReviewDetailAttributes{DemoAccountPassword: migrateRedactionSentinel}
	if !reviewInformationMatches(existing, result.ReviewInformation) {
		t.Fatal("matching against the real password regressed after rendering a redacted copy")
	}
	redacted := asc.AppStoreReviewDetailAttributes{DemoAccountPassword: asc.RedactedValuePlaceholder}
	if reviewInformationMatches(redacted, result.ReviewInformation) {
		t.Fatal("a redacted remote value must not count as a match")
	}
}

func TestPresentableImportResultPassesThroughWhenSensitiveRequested(t *testing.T) {
	password := migrateRedactionSentinel
	result := &MigrateImportResult{ReviewInformation: &ReviewInformation{DemoAccountPassword: &password}}

	if got := presentableImportResult(result, true); got != result {
		t.Fatal("--include-sensitive must render the original result")
	}
	if got := presentableImportResult(nil, false); got != nil {
		t.Fatal("nil result must stay nil")
	}
	if got := presentableImportResult(&MigrateImportResult{}, false); got.ReviewInformation != nil {
		t.Fatal("absent review information must stay absent")
	}
}
