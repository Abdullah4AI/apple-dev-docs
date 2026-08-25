package validation

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// MetadataLengthIssue describes one over-limit metadata field.
type MetadataLengthIssue struct {
	Field  string
	Length int
	Limit  int
	Unit   string
}

// VersionLocalizationLengthIssues returns over-limit fields for one version localization.
func VersionLocalizationLengthIssues(loc VersionLocalization) []MetadataLengthIssue {
	return metadataLengthIssues([]metadataLengthField{
		{field: "description", value: loc.Description, limit: LimitDescription},
		{field: "keywords", value: loc.Keywords, limit: LimitKeywords, length: KeywordFieldLength, unit: keywordLengthUnit},
		{field: "whatsNew", value: loc.WhatsNew, limit: LimitWhatsNew},
		{field: "promotionalText", value: loc.PromotionalText, limit: LimitPromotionalText},
		{field: "marketingUrl", value: loc.MarketingURL, limit: LimitMarketingURL},
		{field: "supportUrl", value: loc.SupportURL, limit: LimitSupportURL},
	})
}

// AppInfoLocalizationLengthIssues returns over-limit fields for one app-info localization.
func AppInfoLocalizationLengthIssues(loc AppInfoLocalization) []MetadataLengthIssue {
	return metadataLengthIssues([]metadataLengthField{
		{field: "name", value: loc.Name, limit: LimitName},
		{field: "subtitle", value: loc.Subtitle, limit: LimitSubtitle},
		{field: "privacyPolicyUrl", value: loc.PrivacyPolicyURL, limit: LimitPrivacyPolicyURL},
		{field: "privacyChoicesUrl", value: loc.PrivacyChoicesURL, limit: LimitPrivacyChoicesURL},
	})
}

// MetadataMinimumLengthIssue describes one metadata field that is too short to
// be real content.
type MetadataMinimumLengthIssue struct {
	Field   string
	Length  int
	Minimum int
}

// VersionLocalizationMinimumLengthIssues returns implausibly short fields for
// one version localization. Empty values are left to the required-field
// checks, which distinguish "unset" from "too short".
func VersionLocalizationMinimumLengthIssues(loc VersionLocalization) []MetadataMinimumLengthIssue {
	return metadataMinimumLengthIssues([]metadataMinimumLengthField{
		{field: "description", value: loc.Description, minimum: MinLengthDescription},
	})
}

// AppInfoLocalizationMinimumLengthIssues returns implausibly short fields for
// one app-info localization.
func AppInfoLocalizationMinimumLengthIssues(loc AppInfoLocalization) []MetadataMinimumLengthIssue {
	return metadataMinimumLengthIssues([]metadataMinimumLengthField{
		{field: "name", value: loc.Name, minimum: MinLengthName},
	})
}

type metadataMinimumLengthField struct {
	field   string
	value   string
	minimum int
}

func metadataMinimumLengthIssues(fields []metadataMinimumLengthField) []MetadataMinimumLengthIssue {
	issues := make([]MetadataMinimumLengthIssue, 0, len(fields))
	for _, field := range fields {
		value := strings.TrimSpace(field.value)
		if value == "" {
			continue
		}
		length := utf8.RuneCountInString(value)
		if length >= field.minimum {
			continue
		}
		issues = append(issues, MetadataMinimumLengthIssue{
			Field:   field.field,
			Length:  length,
			Minimum: field.minimum,
		})
	}
	return issues
}

// MetadataFieldLabel returns the operator-facing label for a metadata field.
func MetadataFieldLabel(field string) string {
	if label, ok := metadataFieldLabels[field]; ok {
		return label
	}
	return field
}

var metadataFieldLabels = map[string]string{
	"whatsNew":          "what's new",
	"promotionalText":   "promotional text",
	"marketingUrl":      "marketing URL",
	"supportUrl":        "support URL",
	"privacyPolicyUrl":  "privacy policy URL",
	"privacyChoicesUrl": "privacy choices URL",
}

// MetadataLengthSeverity returns the severity for an over-limit metadata
// field. URL limits are documented rather than schema-backed, so they stay
// advisory while the text limits Apple publishes per field remain blocking.
func MetadataLengthSeverity(field string) Severity {
	switch field {
	case "marketingUrl", "supportUrl", "privacyPolicyUrl", "privacyChoicesUrl":
		return SeverityWarning
	default:
		return SeverityError
	}
}

type metadataLengthField struct {
	field  string
	value  string
	limit  int
	length func(string) int
	unit   string
}

func metadataLengthIssues(fields []metadataLengthField) []MetadataLengthIssue {
	issues := make([]MetadataLengthIssue, 0, len(fields))
	for _, field := range fields {
		lengthFn := field.length
		if lengthFn == nil {
			lengthFn = utf8.RuneCountInString
		}
		length := lengthFn(field.value)
		if length <= field.limit {
			continue
		}
		unit := field.unit
		if unit == "" {
			unit = "characters"
		}
		issues = append(issues, MetadataLengthIssue{
			Field:  field.field,
			Length: length,
			Limit:  field.limit,
			Unit:   unit,
		})
	}
	return issues
}

func metadataLengthChecks(versionLocs []VersionLocalization, appInfoLocs []AppInfoLocalization) []CheckResult {
	var checks []CheckResult

	for _, loc := range versionLocs {
		for _, issue := range VersionLocalizationLengthIssues(loc) {
			id, ok := versionLengthCheckIDs[issue.Field]
			if !ok {
				continue
			}
			checks = append(checks, metadataLengthCheck(id, issue, loc.Locale, "appStoreVersionLocalization", loc.ID))
		}
	}

	for _, loc := range appInfoLocs {
		for _, issue := range AppInfoLocalizationLengthIssues(loc) {
			id, ok := appInfoLengthCheckIDs[issue.Field]
			if !ok {
				continue
			}
			checks = append(checks, metadataLengthCheck(id, issue, loc.Locale, "appInfoLocalization", loc.ID))
		}
	}

	return checks
}

var versionLengthCheckIDs = map[string]string{
	"description":     "metadata.length.description",
	"keywords":        "metadata.length.keywords",
	"whatsNew":        "metadata.length.whats_new",
	"promotionalText": "metadata.length.promotional_text",
	"marketingUrl":    "metadata.length.marketing_url",
	"supportUrl":      "metadata.length.support_url",
}

var appInfoLengthCheckIDs = map[string]string{
	"name":              "metadata.length.name",
	"subtitle":          "metadata.length.subtitle",
	"privacyPolicyUrl":  "metadata.length.privacy_policy_url",
	"privacyChoicesUrl": "metadata.length.privacy_choices_url",
}

func metadataLengthCheck(id string, issue MetadataLengthIssue, locale, resourceType, resourceID string) CheckResult {
	label := MetadataFieldLabel(issue.Field)
	verb := "exceeds"
	if issue.Field == "keywords" {
		verb = "exceed"
	}

	return CheckResult{
		ID:           id,
		Severity:     MetadataLengthSeverity(issue.Field),
		Locale:       locale,
		Field:        issue.Field,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Message:      fmt.Sprintf("%s %s %d %s", label, verb, issue.Limit, issue.Unit),
		Remediation:  fmt.Sprintf("Shorten %s to %d %s or fewer", label, issue.Limit, issue.Unit),
	}
}
