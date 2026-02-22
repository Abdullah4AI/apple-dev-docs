package validation

import (
	"fmt"
	"strings"
)

// ValidateTestFlight runs TestFlight validation rules and returns a report.
func ValidateTestFlight(input TestFlightInput, strict bool) TestFlightReport {
	checks := make([]CheckResult, 0)
	checks = append(checks, testflightBuildChecks(input.BuildID, input.Build)...)
	checks = append(checks, testflightBuildAppChecks(input.AppID, input.BuildAppID, input.Build)...)
	checks = append(checks, betaReviewDetailsChecks(input.BetaReviewDetails)...)
	checks = append(checks, betaWhatsNewChecks(input.AppPrimaryLocale, input.BetaBuildLocalizations)...)

	summary := summarize(checks, strict)

	buildVersion := ""
	if input.Build != nil {
		buildVersion = strings.TrimSpace(input.Build.Version)
	}

	return TestFlightReport{
		AppID:        input.AppID,
		BuildID:      input.BuildID,
		BuildVersion: buildVersion,
		Summary:      summary,
		Checks:       checks,
		Strict:       strict,
	}
}

func testflightBuildChecks(buildID string, build *Build) []CheckResult {
	buildID = strings.TrimSpace(buildID)
	if build == nil {
		message := "build not found"
		if buildID == "" {
			message = "build is missing"
		}
		return []CheckResult{
			{
				ID:           "testflight.build.missing",
				Severity:     SeverityError,
				ResourceType: "build",
				ResourceID:   buildID,
				Message:      message,
				Remediation:  "Pass a valid build ID with --build",
			},
		}
	}
	return buildChecks(build)
}

func testflightBuildAppChecks(appID string, buildAppID string, build *Build) []CheckResult {
	appID = strings.TrimSpace(appID)
	buildAppID = strings.TrimSpace(buildAppID)
	if build == nil || appID == "" || buildAppID == "" {
		return nil
	}
	if strings.EqualFold(appID, buildAppID) {
		return nil
	}

	return []CheckResult{
		{
			ID:           "testflight.build.app_mismatch",
			Severity:     SeverityError,
			Field:        "app",
			ResourceType: "build",
			ResourceID:   strings.TrimSpace(build.ID),
			Message:      fmt.Sprintf("build belongs to app %s (expected %s)", buildAppID, appID),
			Remediation:  "Use the correct --app for this build (or select a build for the given app)",
		},
	}
}

func betaReviewDetailsChecks(details *BetaReviewDetails) []CheckResult {
	if details == nil {
		return []CheckResult{
			{
				ID:           "testflight.review_details.missing",
				Severity:     SeverityError,
				ResourceType: "betaAppReviewDetail",
				Message:      "beta app review details are missing",
				Remediation:  "Complete TestFlight beta app review details in App Store Connect",
			},
		}
	}

	var checks []CheckResult
	resourceID := strings.TrimSpace(details.ID)

	if strings.TrimSpace(details.ContactFirstName) == "" {
		checks = append(checks, missingBetaReviewDetailsField("contactFirstName", resourceID))
	}
	if strings.TrimSpace(details.ContactLastName) == "" {
		checks = append(checks, missingBetaReviewDetailsField("contactLastName", resourceID))
	}
	if strings.TrimSpace(details.ContactEmail) == "" {
		checks = append(checks, missingBetaReviewDetailsField("contactEmail", resourceID))
	}
	if strings.TrimSpace(details.ContactPhone) == "" {
		checks = append(checks, missingBetaReviewDetailsField("contactPhone", resourceID))
	}

	if details.DemoAccountRequired {
		if strings.TrimSpace(details.DemoAccountName) == "" {
			checks = append(checks, missingBetaReviewDetailsField("demoAccountName", resourceID))
		}
		if strings.TrimSpace(details.DemoAccountPassword) == "" {
			checks = append(checks, missingBetaReviewDetailsField("demoAccountPassword", resourceID))
		}
	}

	return checks
}

func missingBetaReviewDetailsField(field string, resourceID string) CheckResult {
	return CheckResult{
		ID:           "testflight.review_details.missing_field",
		Severity:     SeverityError,
		Field:        field,
		ResourceType: "betaAppReviewDetail",
		ResourceID:   resourceID,
		Message:      "beta app review detail field is missing",
		Remediation:  "Complete TestFlight beta app review details in App Store Connect",
	}
}

func betaWhatsNewChecks(primaryLocale string, localizations []BetaBuildLocalization) []CheckResult {
	primaryLocale = strings.TrimSpace(primaryLocale)

	// Conservative requiredness: ensure at least one localization has "What to Test"
	// populated. We avoid enforcing per-locale completeness, since some apps only
	// ship TestFlight notes in a single language.
	for _, loc := range localizations {
		if strings.TrimSpace(loc.WhatsNew) != "" {
			return nil
		}
	}

	message := `"What to Test" is missing`
	if primaryLocale != "" {
		message = fmt.Sprintf(`"What to Test" is missing (expected at least one localization, e.g. %s)`, primaryLocale)
	}

	return []CheckResult{
		{
			ID:          "testflight.whats_new.missing",
			Severity:    SeverityError,
			Field:       "whatsNew",
			Message:     message,
			Remediation: `Add "What to Test" notes for the build in App Store Connect (TestFlight)`,
		},
	}
}
