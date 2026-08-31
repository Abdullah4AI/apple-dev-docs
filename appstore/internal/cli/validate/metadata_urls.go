package validate

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/metadataurl"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/validation"
)

type validateURLTarget struct {
	RawURL       string
	Locale       string
	Field        string
	ResourceType string
	ResourceID   string
}

var newValidateURLChecker = metadataurl.NewChecker

func validateURLTargets(versionLocs []validation.VersionLocalization, appInfoLocs []validation.AppInfoLocalization) []validateURLTarget {
	targets := make([]validateURLTarget, 0, len(versionLocs)*2+len(appInfoLocs)*2)
	for _, loc := range versionLocs {
		targets = appendValidateURLTarget(targets, loc.SupportURL, loc.Locale, "supportUrl", "appStoreVersionLocalization", loc.ID)
		targets = appendValidateURLTarget(targets, loc.MarketingURL, loc.Locale, "marketingUrl", "appStoreVersionLocalization", loc.ID)
	}
	for _, loc := range appInfoLocs {
		targets = appendValidateURLTarget(targets, loc.PrivacyPolicyURL, loc.Locale, "privacyPolicyUrl", "appInfoLocalization", loc.ID)
		targets = appendValidateURLTarget(targets, loc.PrivacyChoicesURL, loc.Locale, "privacyChoicesUrl", "appInfoLocalization", loc.ID)
	}
	sortValidateURLTargets(targets)
	return targets
}

func appendValidateURLTarget(targets []validateURLTarget, rawURL, locale, field, resourceType, resourceID string) []validateURLTarget {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || !validation.IsValidHTTPURL(rawURL) {
		return targets
	}
	return append(targets, validateURLTarget{
		RawURL:       rawURL,
		Locale:       locale,
		Field:        field,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
}

func checkValidateURLs(ctx context.Context, checker metadataurl.Checker, targets []validateURLTarget) ([]validation.CheckResult, error) {
	urls := make([]string, 0, len(targets))
	for _, target := range targets {
		urls = append(urls, target.RawURL)
	}
	outcomes, err := metadataurl.CheckAll(ctx, checker, urls)
	if err != nil {
		return nil, err
	}

	checks := make([]validation.CheckResult, 0, len(targets))
	for _, target := range targets {
		outcome, ok := outcomes[target.RawURL]
		if !ok {
			outcome.Err = errors.New("metadata URL check did not produce an outcome")
		}
		checks = append(checks, validateURLCheckResults(target, outcome)...)
	}
	sortValidationURLChecks(checks)
	return checks, nil
}

func validateURLCheckResults(target validateURLTarget, outcome metadataurl.Outcome) []validation.CheckResult {
	if outcome.Err != nil {
		id := "legal.url.request_failed"
		message := "declared destination could not be checked"
		remediation := "Verify that the declared destination is reachable from a public network"
		if errors.Is(outcome.Err, metadataurl.ErrUnsafeTarget) {
			id = "legal.url.unsafe_target"
			message = "declared destination could not be reached safely"
			remediation = "Use an HTTP/HTTPS destination on the public internet"
		}
		return []validation.CheckResult{newValidateURLCheck(id, target, message, remediation)}
	}
	if outcome.Result.FinalURL == nil {
		return []validation.CheckResult{newValidateURLCheck(
			"legal.url.request_failed",
			target,
			"declared destination could not be checked",
			"Verify that the declared destination is reachable from a public network",
		)}
	}

	initialURL, err := url.Parse(target.RawURL)
	if err != nil {
		return []validation.CheckResult{newValidateURLCheck(
			"legal.url.request_failed",
			target,
			"declared destination could not be checked",
			"Provide a valid HTTP/HTTPS destination",
		)}
	}

	checks := make([]validation.CheckResult, 0, 2)
	initialHost := strings.ToLower(initialURL.Hostname())
	finalHost := strings.ToLower(outcome.Result.FinalURL.Hostname())
	if outcome.Result.RedirectedHost || (initialHost != "" && finalHost != "" && initialHost != finalHost) {
		checks = append(checks, newValidateURLCheck(
			"legal.url.redirected_host",
			target,
			"declared destination redirects to a different host",
			"Confirm that the redirect destination is intentional and stable",
		))
	}
	if outcome.Result.StatusCode < http.StatusOK || outcome.Result.StatusCode >= http.StatusMultipleChoices {
		checks = append(checks, newValidateURLCheck(
			"legal.url.http_status",
			target,
			"declared destination returned a non-success response",
			"Verify that the declared destination returns a successful HTTP response",
		))
		return checks
	}
	finalPath := outcome.Result.FinalURL.EscapedPath()
	if (finalPath == "" || finalPath == "/") && outcome.Result.FinalURL.RawQuery == "" && outcome.Result.FinalURL.Fragment == "" {
		checks = append(checks, newValidateURLCheck(
			"legal.url.site_root",
			target,
			"declared destination resolves to a site root",
			"Use a dedicated support, marketing, privacy-policy, or privacy-choices page",
		))
	}
	return checks
}

func newValidateURLCheck(id string, target validateURLTarget, message, remediation string) validation.CheckResult {
	return validation.CheckResult{
		ID:           id,
		Severity:     validation.SeverityWarning,
		Locale:       target.Locale,
		Field:        target.Field,
		ResourceType: target.ResourceType,
		ResourceID:   target.ResourceID,
		Message:      message,
		Remediation:  remediation,
	}
}

func sortValidateURLTargets(targets []validateURLTarget) {
	sort.SliceStable(targets, func(i, j int) bool {
		left := targets[i]
		right := targets[j]
		if left.ResourceType != right.ResourceType {
			return left.ResourceType < right.ResourceType
		}
		if left.ResourceID != right.ResourceID {
			return left.ResourceID < right.ResourceID
		}
		if left.Locale != right.Locale {
			return left.Locale < right.Locale
		}
		return left.Field < right.Field
	})
}

func sortValidationURLChecks(checks []validation.CheckResult) {
	sort.SliceStable(checks, func(i, j int) bool {
		left := checks[i]
		right := checks[j]
		if left.ResourceType != right.ResourceType {
			return left.ResourceType < right.ResourceType
		}
		if left.ResourceID != right.ResourceID {
			return left.ResourceID < right.ResourceID
		}
		if left.Locale != right.Locale {
			return left.Locale < right.Locale
		}
		if left.Field != right.Field {
			return left.Field < right.Field
		}
		return left.ID < right.ID
	})
}
