package validation

import "testing"

func TestValidateCarriesVersionStateAndMonetizationEvidence(t *testing.T) {
	report := Validate(Input{
		AppID:                       "app-1",
		VersionID:                   "version-1",
		VersionState:                "PREPARE_FOR_SUBMISSION",
		SubscriptionFetchSkipReason: "required agreement blocked subscriptions",
	}, false)

	if report.VersionState != "PREPARE_FOR_SUBMISSION" {
		t.Fatalf("version state = %q", report.VersionState)
	}
	if report.MonetizationKnown {
		t.Fatal("monetization must remain unknown when subscription evidence was skipped")
	}
}

func TestValidateMarksMonetizationKnownWhenBothCatalogsWereFetched(t *testing.T) {
	report := Validate(Input{AppID: "app-1"}, false)
	if !report.MonetizationKnown {
		t.Fatal("monetization should be known when neither catalog fetch was skipped")
	}
}
