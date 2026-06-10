package submit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/validation"
)

func TestPreflightResult_TallyCounts(t *testing.T) {
	result := &preflightResult{
		Checks: []checkResult{
			{Name: "a", Passed: true},
			{Name: "b", Passed: false},
			{Name: "c", Passed: true},
			{Name: "d", Passed: false},
			{Name: "info", Passed: true, Advisory: true},
		},
	}
	tallyCounts(result)
	if result.PassCount != 3 {
		t.Fatalf("expected 3 passes including advisories, got %d", result.PassCount)
	}
	if result.FailCount != 2 {
		t.Fatalf("expected 2 failures, got %d", result.FailCount)
	}
}

func TestPreflightResult_AllPass(t *testing.T) {
	result := &preflightResult{
		Checks: []checkResult{
			{Name: "a", Passed: true},
			{Name: "b", Passed: true},
			{Name: "info", Passed: true, Advisory: true},
		},
	}
	tallyCounts(result)
	if result.PassCount != 3 {
		t.Fatalf("expected 3 passes including advisories, got %d", result.PassCount)
	}
	if result.FailCount != 0 {
		t.Fatalf("expected 0 failures, got %d", result.FailCount)
	}
}

func TestPrivacyPublishStateAdvisoryCheck_SetsPassedWhenPresent(t *testing.T) {
	check, ok := privacyPublishStateAdvisoryCheck("app-1")
	if !ok {
		t.Fatal("expected advisory check for non-empty app ID")
	}
	if !check.Advisory {
		t.Fatalf("expected advisory flag, got %+v", check)
	}
	if !check.Passed {
		t.Fatalf("expected advisory check to serialize as passed, got %+v", check)
	}
}

func TestPrivacyPublishStateAdvisoryCheck_SkipsBlankAppID(t *testing.T) {
	if _, ok := privacyPublishStateAdvisoryCheck(" \t "); ok {
		t.Fatal("expected blank app ID to skip advisory check")
	}
}

func TestPreflightResultFromReport_MapsContentRightsCheckName(t *testing.T) {
	result := preflightResultFromReport("app-123", "1.0", validation.Report{
		Platform: "IOS",
		Checks: []validation.CheckResult{
			{
				ID:       "content_rights.missing",
				Severity: validation.SeverityError,
				Message:  "content rights declaration is not set",
			},
		},
	})

	if len(result.Checks) != 1 {
		t.Fatalf("expected one check, got %+v", result.Checks)
	}
	if result.Checks[0].Name != "Content rights" {
		t.Fatalf("expected content rights label, got %+v", result.Checks[0])
	}
}

func TestPreflightTextOutput(t *testing.T) {
	var buf bytes.Buffer
	printPreflightText(&buf, &preflightResult{
		AppID:    "123",
		Version:  "1.0",
		Platform: "IOS",
		Checks: []checkResult{
			{Name: "Version exists", Passed: true, Message: "Version 1.0 found"},
			{Name: "Build attached", Passed: false, Message: "No build", Hint: "Attach a build with `asc release stage ...`, or upload and submit with `asc publish appstore ... --submit`"},
			{Name: "App Privacy", Advisory: true, Message: "App Privacy publish state is not verifiable via the public App Store Connect API and may still block submission", Hint: "Confirm App Privacy is published in App Store Connect before submitting: https://appstoreconnect.apple.com/apps/123/appPrivacy"},
		},
		PassCount: 1,
		FailCount: 1,
	})
	if !strings.Contains(buf.String(), "Preflight check for app 123 v1.0 (IOS)") {
		t.Fatalf("expected header in text output, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "App Privacy publish state is not verifiable via the public App Store Connect API") {
		t.Fatalf("expected advisory in text output, got %q", buf.String())
	}
}

func TestPreflightTextOutput_AdvisoryOnlyDoesNotClaimReadyToSubmit(t *testing.T) {
	var buf bytes.Buffer
	printPreflightText(&buf, &preflightResult{
		AppID:    "123",
		Version:  "1.0",
		Platform: "IOS",
		Checks: []checkResult{
			{
				Name:     "App Privacy",
				Passed:   true,
				Advisory: true,
				Message:  "App Privacy publish state is not verifiable via the public App Store Connect API and may still block submission",
				Hint:     "Confirm App Privacy is published in App Store Connect before submitting: https://appstoreconnect.apple.com/apps/123/appPrivacy",
			},
		},
	})

	output := buf.String()
	if strings.Contains(output, "Ready to submit") {
		t.Fatalf("did not expect advisory-only result to claim readiness, got %q", output)
	}
	if !strings.Contains(output, "Result: Required checks passed, but 1 advisory should be reviewed before submitting.") {
		t.Fatalf("expected advisory summary in text output, got %q", output)
	}
}
