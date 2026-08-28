package validation

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Placeholder-content lint is deliberately narrow. It flags unmistakable
// template copy while leaving product, platform, roadmap, and beta wording to
// App Review instead of guessing at editorial intent.
var placeholderPhrases = []string{
	"lorem ipsum dolor sit amet",
}

// These markers are case-sensitive because their lowercase spellings can be
// ordinary product vocabulary, especially "todo" applications.
var placeholderMarkers = []string{"TODO", "TBD", "FIXME"}

var placeholderPatterns = []*regexp.Regexp{
	contentPhrasePattern(placeholderPhrases),
	contentMarkerPattern(placeholderMarkers),
}

type contentField struct {
	field        string
	label        string
	value        string
	locale       string
	resourceType string
	resourceID   string
}

type contentMatch struct {
	start int
	end   int
}

func contentChecks(versionLocs []VersionLocalization, appInfoLocs []AppInfoLocalization) []CheckResult {
	fields := make([]contentField, 0, len(versionLocs)*4+len(appInfoLocs)*2)

	for _, loc := range versionLocs {
		base := contentField{locale: loc.Locale, resourceType: "appStoreVersionLocalization", resourceID: loc.ID}
		fields = append(
			fields,
			contentFieldWith(base, "description", "description", loc.Description),
			contentFieldWith(base, "keywords", "keywords", loc.Keywords),
			contentFieldWith(base, "whatsNew", "what's new", loc.WhatsNew),
			contentFieldWith(base, "promotionalText", "promotional text", loc.PromotionalText),
		)
	}

	for _, loc := range appInfoLocs {
		base := contentField{locale: loc.Locale, resourceType: "appInfoLocalization", resourceID: loc.ID}
		fields = append(
			fields,
			contentFieldWith(base, "name", "name", loc.Name),
			contentFieldWith(base, "subtitle", "subtitle", loc.Subtitle),
		)
	}

	var checks []CheckResult
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			continue
		}
		matches := findContentMatches(field.value, placeholderPatterns)
		if len(matches) == 0 {
			continue
		}
		checks = append(checks, CheckResult{
			ID:           "content.placeholder_text",
			Severity:     SeverityWarning,
			Locale:       field.locale,
			Field:        field.field,
			ResourceType: field.resourceType,
			ResourceID:   field.resourceID,
			Message:      fmt.Sprintf("%s contains placeholder text (%s)", field.label, quoteContentMatches(matches)),
			Remediation:  "Replace the placeholder copy with the final listing text",
		})
	}

	return checks
}

func contentFieldWith(base contentField, field string, label string, value string) contentField {
	base.field = field
	base.label = label
	base.value = value
	return base
}

// findContentMatches returns distinct normalized phrases in source order.
func findContentMatches(value string, patterns []*regexp.Regexp) []string {
	ranges := make([]contentMatch, 0, 2)
	for _, pattern := range patterns {
		for _, location := range pattern.FindAllStringIndex(value, -1) {
			if !hasContentTokenBoundaries(value, location[0], location[1]) {
				continue
			}
			ranges = append(ranges, contentMatch{start: location[0], end: location[1]})
		}
	}
	sort.SliceStable(ranges, func(i, j int) bool {
		if ranges[i].start != ranges[j].start {
			return ranges[i].start < ranges[j].start
		}
		return ranges[i].end > ranges[j].end
	})

	seen := make(map[string]struct{})
	matches := make([]string, 0, len(ranges))
	for _, location := range ranges {
		phrase := strings.Join(strings.Fields(value[location.start:location.end]), " ")
		if !shouldReportContentMatch(value, location, phrase) {
			continue
		}
		key := strings.ToLower(phrase)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		matches = append(matches, phrase)
	}
	return matches
}

// shouldReportContentMatch requires marker punctuation for TODO because the
// same uppercase word is ordinary localized copy and can be a product name.
// TBD and FIXME remain unambiguous standalone markers.
func shouldReportContentMatch(value string, location contentMatch, phrase string) bool {
	if phrase != "TODO" {
		return true
	}
	end := location.end
	for end < len(value) {
		runeValue, size := utf8.DecodeRuneInString(value[end:])
		if unicode.IsSpace(runeValue) {
			end += size
			continue
		}
		return runeValue == ':' || runeValue == '\uff1a' || runeValue == '-' || runeValue == '\u2013' || runeValue == '\u2014'
	}
	return false
}

func quoteContentMatches(matches []string) string {
	quoted := make([]string, 0, len(matches))
	for _, match := range matches {
		quoted = append(quoted, fmt.Sprintf("%q", match))
	}
	return strings.Join(quoted, ", ")
}

func contentPhrasePattern(phrases []string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(?:` + contentAlternation(phrases) + `)`)
}

func contentMarkerPattern(markers []string) *regexp.Regexp {
	return regexp.MustCompile(`(?:` + contentAlternation(markers) + `)`)
}

func hasContentTokenBoundaries(value string, start, end int) bool {
	if start > 0 {
		previous, _ := utf8.DecodeLastRuneInString(value[:start])
		if isContentTokenRune(previous) {
			return false
		}
	}
	if end < len(value) {
		next, _ := utf8.DecodeRuneInString(value[end:])
		if isContentTokenRune(next) {
			return false
		}
	}
	return true
}

func isContentTokenRune(value rune) bool {
	return value == '_' || unicode.IsLetter(value) || unicode.IsNumber(value) || unicode.IsMark(value)
}

func contentAlternation(phrases []string) string {
	ordered := append([]string(nil), phrases...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return len(ordered[i]) > len(ordered[j])
	})

	parts := make([]string, 0, len(ordered))
	for _, phrase := range ordered {
		words := strings.Fields(phrase)
		escaped := make([]string, 0, len(words))
		for _, word := range words {
			escaped = append(escaped, regexp.QuoteMeta(word))
		}
		parts = append(parts, strings.Join(escaped, `[\s\v\x{0085}\p{Z}]+`))
	}
	return strings.Join(parts, "|")
}
