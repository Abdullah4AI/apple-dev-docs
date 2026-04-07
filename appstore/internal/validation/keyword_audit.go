package validation

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const keywordAuditUnderfilledRemainingCharacters = 25

// KeywordAuditInput describes the inputs for a keyword audit report.
type KeywordAuditInput struct {
	AppID                string
	VersionID            string
	VersionString        string
	Platform             string
	BlockedTerms         []string
	VersionLocalizations []VersionLocalization
	AppInfoLocalizations []AppInfoLocalization
}

// KeywordAuditLocale summarizes one audited locale.
type KeywordAuditLocale struct {
	Locale                string `json:"locale"`
	VersionLocalizationID string `json:"versionLocalizationId,omitempty"`
	AppInfoLocalizationID string `json:"appInfoLocalizationId,omitempty"`
	KeywordField          string `json:"keywordField,omitempty"`
	KeywordCount          int    `json:"keywordCount"`
	// Legacy JSON field names are retained for compatibility with existing audit consumers.
	UsedCharacters      int    `json:"usedBytes"`
	RemainingCharacters int    `json:"remainingBytes"`
	Name                string `json:"name,omitempty"`
	Subtitle            string `json:"subtitle,omitempty"`
	Errors              int    `json:"errors"`
	Warnings            int    `json:"warnings"`
	Infos               int    `json:"infos"`
}

// KeywordAuditCheck represents one keyword-audit finding.
type KeywordAuditCheck struct {
	ID                  string   `json:"id"`
	Severity            Severity `json:"severity"`
	Message             string   `json:"message"`
	Remediation         string   `json:"remediation,omitempty"`
	Locale              string   `json:"locale,omitempty"`
	Field               string   `json:"field,omitempty"`
	Keyword             string   `json:"keyword,omitempty"`
	MatchedTerm         string   `json:"matchedTerm,omitempty"`
	RelatedLocales      []string `json:"relatedLocales,omitempty"`
	UsedCharacters      int      `json:"usedBytes,omitempty"`
	RemainingCharacters int      `json:"remainingBytes,omitempty"`
}

// KeywordAuditReport is the top-level keyword-audit output.
type KeywordAuditReport struct {
	AppID         string               `json:"appId"`
	VersionID     string               `json:"versionId"`
	VersionString string               `json:"versionString,omitempty"`
	Platform      string               `json:"platform,omitempty"`
	BlockedTerms  []string             `json:"blockedTerms,omitempty"`
	Summary       Summary              `json:"summary"`
	Locales       []KeywordAuditLocale `json:"locales"`
	Checks        []KeywordAuditCheck  `json:"checks"`
	Strict        bool                 `json:"strict,omitempty"`
}

type keywordFieldScan struct {
	tokens                 []string
	emptySegments          bool
	nonCanonicalSeparators bool
}

type crossLocaleKeywordAuditEntry struct {
	locales map[string]struct{}
	phrases map[string]struct{}
}

// AuditKeywords builds a keyword-quality report across version localizations.
func AuditKeywords(input KeywordAuditInput, strict bool) KeywordAuditReport {
	report := KeywordAuditReport{
		AppID:         strings.TrimSpace(input.AppID),
		VersionID:     strings.TrimSpace(input.VersionID),
		VersionString: strings.TrimSpace(input.VersionString),
		Platform:      strings.TrimSpace(input.Platform),
		BlockedTerms:  NormalizeKeywordAuditTerms(input.BlockedTerms),
		Locales:       make([]KeywordAuditLocale, 0, len(input.VersionLocalizations)),
		Checks:        make([]KeywordAuditCheck, 0),
		Strict:        strict,
	}

	appInfoByLocale := make(map[string]AppInfoLocalization, len(input.AppInfoLocalizations))
	for _, loc := range input.AppInfoLocalizations {
		if locale := strings.TrimSpace(loc.Locale); locale != "" {
			appInfoByLocale[locale] = loc
		}
	}

	localeSummaries := make(map[string]*KeywordAuditLocale, len(input.VersionLocalizations))
	crossLocaleKeywords := make(map[string]*crossLocaleKeywordAuditEntry)

	for _, loc := range input.VersionLocalizations {
		locale := strings.TrimSpace(loc.Locale)
		if locale == "" {
			continue
		}

		scan := scanKeywordField(loc.Keywords)
		normalized, duplicates := normalizeKeywordAuditTokens(scan.tokens)
		usedCharacters := KeywordFieldLength(loc.Keywords)
		remainingCharacters := LimitKeywords - usedCharacters
		if remainingCharacters < 0 {
			remainingCharacters = 0
		}

		summary := &KeywordAuditLocale{
			Locale:                locale,
			VersionLocalizationID: strings.TrimSpace(loc.ID),
			KeywordField:          loc.Keywords,
			KeywordCount:          len(normalized),
			UsedCharacters:        usedCharacters,
			RemainingCharacters:   remainingCharacters,
		}
		if appInfo, ok := appInfoByLocale[locale]; ok {
			summary.AppInfoLocalizationID = strings.TrimSpace(appInfo.ID)
			summary.Name = appInfo.Name
			summary.Subtitle = appInfo.Subtitle
		}
		localeSummaries[locale] = summary

		if issue := KeywordFieldLengthIssue(loc.Keywords); issue != nil {
			report.Checks = append(report.Checks, KeywordAuditCheck{
				ID:                  "metadata.keywords.length",
				Severity:            SeverityError,
				Locale:              locale,
				Field:               "keywords",
				Message:             fmt.Sprintf("keywords exceed %d %s", issue.Limit, issue.Unit),
				Remediation:         fmt.Sprintf("Shorten keywords to %d %s or fewer", issue.Limit, issue.Unit),
				UsedCharacters:      issue.Length,
				RemainingCharacters: 0,
			})
		}

		if scan.emptySegments {
			report.Checks = append(report.Checks, KeywordAuditCheck{
				ID:          "metadata.keywords.empty_segments",
				Severity:    SeverityWarning,
				Locale:      locale,
				Field:       "keywords",
				Message:     "keyword field contains empty phrase segments",
				Remediation: "Remove repeated, leading, or trailing separators from the keyword field",
			})
		}
		if scan.nonCanonicalSeparators {
			report.Checks = append(report.Checks, KeywordAuditCheck{
				ID:          "metadata.keywords.noncanonical_separators",
				Severity:    SeverityWarning,
				Locale:      locale,
				Field:       "keywords",
				Message:     "keyword field uses non-canonical separators",
				Remediation: "Use a comma-separated keyword field",
			})
		}

		if len(duplicates) > 0 {
			report.Checks = append(report.Checks, KeywordAuditCheck{
				ID:          "metadata.keywords.locale_duplicates",
				Severity:    SeverityWarning,
				Locale:      locale,
				Field:       "keywords",
				Keyword:     strings.Join(duplicates, ", "),
				Message:     fmt.Sprintf("keywords repeat %d phrase(s) within the locale", len(duplicates)),
				Remediation: "Remove duplicated phrases from the keyword field",
			})
		}

		if summary.RemainingCharacters >= keywordAuditUnderfilledRemainingCharacters {
			report.Checks = append(report.Checks, KeywordAuditCheck{
				ID:                  "metadata.keywords.underfilled",
				Severity:            SeverityInfo,
				Locale:              locale,
				Field:               "keywords",
				Message:             fmt.Sprintf("keyword field leaves %d characters unused", summary.RemainingCharacters),
				Remediation:         "Consider using more of the keyword budget if the missing space is intentional and safe",
				UsedCharacters:      summary.UsedCharacters,
				RemainingCharacters: summary.RemainingCharacters,
			})
		}

		if appInfo, ok := appInfoByLocale[locale]; ok {
			nameText := normalizeKeywordAuditText(appInfo.Name)
			subtitleText := normalizeKeywordAuditText(appInfo.Subtitle)
			for _, phrase := range normalized {
				phraseText := normalizeKeywordAuditText(phrase)
				if phraseText == "" {
					continue
				}
				if containsNormalizedPhrase(nameText, phraseText) {
					report.Checks = append(report.Checks, KeywordAuditCheck{
						ID:          "metadata.keywords.overlap_name",
						Severity:    SeverityWarning,
						Locale:      locale,
						Field:       "name",
						Keyword:     phrase,
						Message:     fmt.Sprintf("keyword phrase %q overlaps the localized app name", phrase),
						Remediation: "Avoid repeating name terms inside the keyword field",
					})
				}
				if containsNormalizedPhrase(subtitleText, phraseText) {
					report.Checks = append(report.Checks, KeywordAuditCheck{
						ID:          "metadata.keywords.overlap_subtitle",
						Severity:    SeverityWarning,
						Locale:      locale,
						Field:       "subtitle",
						Keyword:     phrase,
						Message:     fmt.Sprintf("keyword phrase %q overlaps the localized subtitle", phrase),
						Remediation: "Avoid repeating subtitle terms inside the keyword field",
					})
				}
			}
		}

		for _, phrase := range normalized {
			phraseText := normalizeKeywordAuditText(phrase)
			if phraseText == "" {
				continue
			}

			entry := crossLocaleKeywords[phraseText]
			if entry == nil {
				entry = &crossLocaleKeywordAuditEntry{
					locales: make(map[string]struct{}),
					phrases: make(map[string]struct{}),
				}
				crossLocaleKeywords[phraseText] = entry
			}
			entry.locales[locale] = struct{}{}
			entry.phrases[phrase] = struct{}{}

			for _, term := range report.BlockedTerms {
				termText := normalizeKeywordAuditText(term)
				if termText == "" || !containsNormalizedPhrase(phraseText, termText) {
					continue
				}
				report.Checks = append(report.Checks, KeywordAuditCheck{
					ID:          "metadata.keywords.blocked_term",
					Severity:    SeverityWarning,
					Locale:      locale,
					Field:       "keywords",
					Keyword:     phrase,
					MatchedTerm: term,
					Message:     fmt.Sprintf("keyword phrase %q matches blocked term %q", phrase, term),
					Remediation: "Remove or replace the blocked phrase",
				})
			}
		}
	}

	if len(localeSummaries) == 0 {
		report.Checks = append(report.Checks, KeywordAuditCheck{
			ID:          "metadata.keywords.localizations_missing",
			Severity:    SeverityError,
			Field:       "keywords",
			Message:     "no version localizations were available for keyword audit",
			Remediation: "Create or fetch at least one version localization before auditing keywords",
		})
	}

	for _, entry := range crossLocaleKeywords {
		if len(entry.locales) < 2 {
			continue
		}
		related := make([]string, 0, len(entry.locales))
		for locale := range entry.locales {
			related = append(related, locale)
		}
		sort.Strings(related)
		displayPhrases := make([]string, 0, len(entry.phrases))
		for phrase := range entry.phrases {
			displayPhrases = append(displayPhrases, phrase)
		}
		sort.Slice(displayPhrases, func(i, j int) bool {
			left := strings.ToLower(displayPhrases[i])
			right := strings.ToLower(displayPhrases[j])
			if left != right {
				return left < right
			}
			return displayPhrases[i] < displayPhrases[j]
		})
		displayPhrase := displayPhrases[0]
		report.Checks = append(report.Checks, KeywordAuditCheck{
			ID:             "metadata.keywords.cross_locale_duplicates",
			Severity:       SeverityInfo,
			Field:          "keywords",
			Keyword:        displayPhrase,
			RelatedLocales: related,
			Message:        fmt.Sprintf("keyword phrase %q appears in multiple locales", displayPhrase),
			Remediation:    "Confirm the repeated phrase is intentional across the listed locales",
		})
	}

	sort.Slice(report.Checks, func(i, j int) bool {
		left := report.Checks[i]
		right := report.Checks[j]
		if left.Locale != right.Locale {
			return left.Locale < right.Locale
		}
		if left.ID != right.ID {
			return left.ID < right.ID
		}
		if left.Keyword != right.Keyword {
			return left.Keyword < right.Keyword
		}
		return left.Message < right.Message
	})

	locales := make([]string, 0, len(localeSummaries))
	for locale := range localeSummaries {
		locales = append(locales, locale)
	}
	sort.Strings(locales)

	for _, check := range report.Checks {
		incrementKeywordAuditSeverity(localeSummaries[check.Locale], check.Severity)
		for _, locale := range check.RelatedLocales {
			if locale == check.Locale {
				continue
			}
			incrementKeywordAuditSeverity(localeSummaries[locale], check.Severity)
		}
	}

	for _, locale := range locales {
		report.Locales = append(report.Locales, *localeSummaries[locale])
	}

	report.Summary = summarizeKeywordAudit(report.Checks, strict)
	return report
}

func summarizeKeywordAudit(checks []KeywordAuditCheck, strict bool) Summary {
	summary := Summary{}
	for _, check := range checks {
		switch check.Severity {
		case SeverityError:
			summary.Errors++
		case SeverityWarning:
			summary.Warnings++
		case SeverityInfo:
			summary.Infos++
		}
	}
	summary.Blocking = summary.Errors
	if strict {
		summary.Blocking += summary.Warnings
	}
	return summary
}

func incrementKeywordAuditSeverity(locale *KeywordAuditLocale, severity Severity) {
	if locale == nil {
		return
	}
	switch severity {
	case SeverityError:
		locale.Errors++
	case SeverityWarning:
		locale.Warnings++
	case SeverityInfo:
		locale.Infos++
	}
}

// NormalizeKeywordAuditTerms trims, de-duplicates, and sorts user-supplied terms.
func NormalizeKeywordAuditTerms(terms []string) []string {
	normalized := make([]string, 0, len(terms))
	seen := make(map[string]struct{}, len(terms))
	for _, term := range terms {
		trimmed := strings.Join(strings.Fields(strings.TrimSpace(term)), " ")
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Slice(normalized, func(i, j int) bool {
		return strings.ToLower(normalized[i]) < strings.ToLower(normalized[j])
	})
	return normalized
}

func scanKeywordField(value string) keywordFieldScan {
	scan := keywordFieldScan{
		tokens: make([]string, 0),
	}

	var current strings.Builder
	flush := func(separator rune) {
		token := strings.Join(strings.Fields(strings.TrimSpace(current.String())), " ")
		if token == "" {
			if len(scan.tokens) > 0 || current.Len() > 0 || separator == ',' {
				scan.emptySegments = true
			}
		} else {
			scan.tokens = append(scan.tokens, token)
		}
		current.Reset()
	}

	for _, r := range value {
		switch r {
		case ',':
			flush(r)
		case '，', '、', ';', '；', '\n', '\r':
			scan.nonCanonicalSeparators = true
			flush(r)
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		flush(0)
	} else if strings.HasSuffix(value, ",") {
		scan.emptySegments = true
	}

	return scan
}

func normalizeKeywordAuditTokens(tokens []string) ([]string, []string) {
	normalized := make([]string, 0, len(tokens))
	duplicates := make([]string, 0)
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		trimmed := strings.Join(strings.Fields(strings.TrimSpace(token)), " ")
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			duplicates = append(duplicates, trimmed)
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized, duplicates
}

func normalizeKeywordAuditText(value string) string {
	var builder strings.Builder
	lastSpace := false
	for _, r := range strings.TrimSpace(value) {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r):
			builder.WriteRune(unicode.ToLower(r))
			lastSpace = false
		default:
			if !lastSpace && builder.Len() > 0 {
				builder.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

func containsNormalizedPhrase(haystack string, needle string) bool {
	haystack = strings.TrimSpace(haystack)
	needle = strings.TrimSpace(needle)
	if haystack == "" || needle == "" {
		return false
	}
	return strings.Contains(" "+haystack+" ", " "+needle+" ")
}
