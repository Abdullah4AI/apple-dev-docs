package shared

import (
	"sort"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

// SubmitReadinessIssue describes submission-blocking missing fields for a locale.
type SubmitReadinessIssue struct {
	Locale        string
	MissingFields []string
}

// MissingSubmitRequiredLocalizationFields returns missing metadata fields that
// block App Store submission for a version localization.
func MissingSubmitRequiredLocalizationFields(attrs asc.AppStoreVersionLocalizationAttributes) []string {
	missing := make([]string, 0, 3)
	if strings.TrimSpace(attrs.Description) == "" {
		missing = append(missing, "description")
	}
	if strings.TrimSpace(attrs.Keywords) == "" {
		missing = append(missing, "keywords")
	}
	if strings.TrimSpace(attrs.SupportURL) == "" {
		missing = append(missing, "supportUrl")
	}
	return missing
}

// SubmitReadinessIssuesByLocale evaluates all localizations and returns
// per-locale missing submit-required fields.
func SubmitReadinessIssuesByLocale(localizations []asc.Resource[asc.AppStoreVersionLocalizationAttributes]) []SubmitReadinessIssue {
	issues := make([]SubmitReadinessIssue, 0, len(localizations))
	for _, localization := range localizations {
		missing := MissingSubmitRequiredLocalizationFields(localization.Attributes)
		if len(missing) == 0 {
			continue
		}

		locale := strings.TrimSpace(localization.Attributes.Locale)
		if locale == "" {
			locale = "<unknown>"
		}
		issues = append(issues, SubmitReadinessIssue{
			Locale:        locale,
			MissingFields: missing,
		})
	}

	sort.SliceStable(issues, func(i, j int) bool {
		return issues[i].Locale < issues[j].Locale
	})
	return issues
}
