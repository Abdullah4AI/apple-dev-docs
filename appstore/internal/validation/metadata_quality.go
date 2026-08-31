package validation

import "sort"

// MetadataQualityChecks returns deterministic, warning-only checks that can be
// evaluated from the localized metadata already loaded by asc validate.
//
// It intentionally projects only the narrow, actionable subset of the keyword
// audit. Informational budget/cross-locale findings, user-supplied blocked
// terms, and the audit's missing-localization error remain owned by the
// dedicated keyword audit command.
func MetadataQualityChecks(versionLocs []VersionLocalization, appInfoLocs []AppInfoLocalization) []CheckResult {
	checks := make([]CheckResult, 0)

	for _, loc := range appInfoLocs {
		for _, issue := range AppInfoLocalizationMinimumLengthIssues(loc) {
			if issue.Field != "name" {
				continue
			}
			checks = append(checks, CheckResult{
				ID:           "metadata.minimum.name",
				Severity:     SeverityWarning,
				Locale:       loc.Locale,
				Field:        issue.Field,
				ResourceType: "appInfoLocalization",
				ResourceID:   loc.ID,
				Message:      "app name is shorter than 2 characters",
				Remediation:  "Use an app name with at least 2 characters",
			})
		}
	}

	keywordChecks := AuditKeywords(KeywordAuditInput{
		VersionLocalizations: versionLocs,
		AppInfoLocalizations: appInfoLocs,
	}, false).Checks
	versionByLocale := make(map[string]VersionLocalization, len(versionLocs))
	for _, loc := range versionLocs {
		locale := loc.Locale
		if current, exists := versionByLocale[locale]; !exists || loc.ID < current.ID {
			versionByLocale[locale] = loc
		}
	}

	allowed := map[string]struct{}{
		"metadata.keywords.empty_segments":          {},
		"metadata.keywords.noncanonical_separators": {},
		"metadata.keywords.locale_duplicates":       {},
		"metadata.keywords.overlap_name":            {},
		"metadata.keywords.overlap_subtitle":        {},
	}
	messages := map[string][2]string{
		"metadata.keywords.empty_segments": {
			"keyword field contains empty phrase segments",
			"Remove repeated, leading, or trailing separators from the keyword field",
		},
		"metadata.keywords.noncanonical_separators": {
			"keyword field uses non-canonical separators",
			"Use a comma-separated keyword field",
		},
		"metadata.keywords.locale_duplicates": {
			"keyword field repeats one or more phrases within this locale",
			"Remove duplicated phrases from the keyword field",
		},
		"metadata.keywords.overlap_name": {
			"keyword field repeats one or more localized app-name terms",
			"Avoid repeating app-name terms inside the keyword field",
		},
		"metadata.keywords.overlap_subtitle": {
			"keyword field repeats one or more localized subtitle terms",
			"Avoid repeating subtitle terms inside the keyword field",
		},
	}
	seen := make(map[string]struct{}, len(keywordChecks))
	for _, keywordCheck := range keywordChecks {
		if _, ok := allowed[keywordCheck.ID]; !ok {
			continue
		}
		loc, ok := versionByLocale[keywordCheck.Locale]
		if !ok {
			continue
		}
		key := keywordCheck.Locale + "\x00" + keywordCheck.ID
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		message := messages[keywordCheck.ID]
		checks = append(checks, CheckResult{
			ID:           keywordCheck.ID,
			Severity:     SeverityWarning,
			Locale:       keywordCheck.Locale,
			Field:        "keywords",
			ResourceType: "appStoreVersionLocalization",
			ResourceID:   loc.ID,
			Message:      message[0],
			Remediation:  message[1],
		})
	}

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

	return checks
}
