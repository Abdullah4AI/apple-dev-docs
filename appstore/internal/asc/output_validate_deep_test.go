package asc

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/validation"
)

func TestValidationRowsAddResolutionColumnsOnlyForDeepReports(t *testing.T) {
	base := &validation.Report{Checks: []validation.CheckResult{{ID: "check-1", Severity: validation.SeverityError, Message: "broken", Remediation: "repair"}}}
	headers, _ := validationCheckRows(base)
	if slices.Contains(headers, "Fixability") {
		t.Fatalf("default validation headers changed: %#v", headers)
	}
	defaultMarkdown := captureStdout(t, func() error { return PrintMarkdown(base) })
	if strings.Contains(defaultMarkdown, "Fixability") || strings.Contains(defaultMarkdown, "App Store Connect URL") {
		t.Fatalf("default Markdown gained deep-only columns:\n%s", defaultMarkdown)
	}

	base.Deep = &validation.DeepReport{}
	base.Checks[0].Resolution = &validation.Resolution{
		Fixability:         validation.FixabilityWebFixable,
		Commands:           []string{"asc web repair --confirm"},
		AppStoreConnectURL: "https://appstoreconnect.apple.com/apps/app-1",
	}
	headers, rows := validationCheckRows(base)
	for _, want := range []string{"Fixability", "Commands", "App Store Connect URL"} {
		if !slices.Contains(headers, want) {
			t.Fatalf("deep headers %#v missing %q", headers, want)
		}
	}
	if got := rows[0][len(rows[0])-3]; got != "web-fixable" {
		t.Fatalf("fixability cell = %q", got)
	}
	deepMarkdown := captureStdout(t, func() error { return PrintMarkdown(base) })
	for _, want := range []string{"Fixability", "Commands", "App Store Connect URL", "web-fixable", "asc web repair --confirm"} {
		if !strings.Contains(deepMarkdown, want) {
			t.Fatalf("deep Markdown missing %q:\n%s", want, deepMarkdown)
		}
	}

	deepHeaders, deepRows := validationDeepRows(base)
	if len(deepHeaders) == 0 || len(deepRows) == 0 {
		t.Fatalf("deep rows were not rendered: headers=%#v rows=%#v", deepHeaders, deepRows)
	}
}

func TestValidationMetadataFindingsRenderAcrossOutputFormats(t *testing.T) {
	report := &validation.Report{
		AppID:     "app-1",
		VersionID: "version-1",
		Checks: []validation.CheckResult{
			{
				ID:           "metadata.minimum.name",
				Severity:     validation.SeverityWarning,
				Locale:       "en-US",
				Field:        "name",
				ResourceType: "appInfoLocalization",
				ResourceID:   "app-info-loc-1",
				Message:      "app name is shorter than 2 characters",
				Remediation:  "Use an app name with at least 2 characters",
			},
			{
				ID:           "legal.url.http_status",
				Severity:     validation.SeverityWarning,
				Locale:       "en-US",
				Field:        "supportUrl",
				ResourceType: "appStoreVersionLocalization",
				ResourceID:   "version-loc-1",
				Message:      "declared destination returned a non-success response",
				Remediation:  "Verify that the declared destination returns a successful HTTP response",
			},
		},
	}

	jsonOutput := captureStdout(t, func() error { return PrintJSON(report) })
	var decoded validation.Report
	if err := json.Unmarshal([]byte(jsonOutput), &decoded); err != nil {
		t.Fatalf("JSON output error: %v; output=%q", err, jsonOutput)
	}
	if len(decoded.Checks) != 2 || decoded.Checks[0].ID != "metadata.minimum.name" || decoded.Checks[1].ID != "legal.url.http_status" {
		t.Fatalf("decoded checks = %+v, want both metadata findings", decoded.Checks)
	}

	for _, renderer := range []struct {
		name string
		fn   func(any) error
	}{{name: "table", fn: PrintTable}, {name: "markdown", fn: PrintMarkdown}} {
		t.Run(renderer.name, func(t *testing.T) {
			output := captureStdout(t, func() error { return renderer.fn(report) })
			for _, want := range []string{"metadata.minimum.name", "legal.url.http_status", "supportUrl", "non-success response"} {
				if !strings.Contains(output, want) {
					t.Fatalf("%s output missing %q: %s", renderer.name, want, output)
				}
			}
		})
	}
}
