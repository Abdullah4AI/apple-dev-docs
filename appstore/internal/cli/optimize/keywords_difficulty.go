// Package optimize composes official Apple sources into read-only
// optimization workflows.
//
// The keyword difficulty methodology implemented in this file — the signal
// weights and constants, the keyword match ladder, and the brand heuristic —
// is adapted from semihcihan's App Store Optimization CLI (MIT licensed):
// https://github.com/semihcihan/App-Store-Optimization-CLI
//
// This is an independent Go implementation of that published formula rather
// than ported code. docs/design/optimize-keywords.md documents the formula,
// its named inputs, and its limitations.
package optimize

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"golang.org/x/text/unicode/norm"
)

// Keyword match ladder rungs, ordered from strongest to weakest evidence that
// an app is deliberately targeting a keyword.
const (
	keywordMatchTitleExactPhrase    = "titleExactPhrase"
	keywordMatchTitleAllWords       = "titleAllWords"
	keywordMatchSubtitleExactPhrase = "subtitleExactPhrase"
	keywordMatchCombinedPhrase      = "combinedPhrase"
	keywordMatchSubtitleAllWords    = "subtitleAllWords"
	keywordMatchNone                = "none"
)

// Difficulty engine constants. Every value is documented in
// docs/design/optimize-keywords.md so a score can be re-derived by hand.
const (
	keywordRatingCountCeiling  = 10000
	keywordRatingFloor         = 3.0
	keywordRatingConfidenceMin = 20
	keywordAgeWindowDays       = 365.0
	keywordMissingDateDays     = 365.0
	keywordRatingsPerDayCeil   = 100.0

	keywordWeightRatingCount   = 0.2
	keywordWeightAverageRating = 0.1
	keywordWeightAge           = 0.1
	keywordWeightKeywordMatch  = 0.3
	keywordWeightRatingsPerDay = 0.3

	keywordAppCountFloor  = 10
	keywordAppCountCeil   = 200
	keywordDifficultyNorm = 6.5
	keywordFallbackApps   = 5

	keywordBrandLeaderRatings = 1000
	keywordBrandPeerRatings   = 10000

	keywordCompetitorSampleSize = 5
)

var keywordBrandTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

// competitorApp is the raw public metadata behind one competitor's app score.
type competitorApp struct {
	AppID                     string
	Name                      string
	Subtitle                  string
	PublisherName             string
	AverageUserRating         float64
	UserRatingCount           int64
	ReleaseDate               string
	CurrentVersionReleaseDate string
}

// keywordDifficultyResult is the keyword-level aggregation of competitor app
// scores, carrying its own inputs so the difficulty stays reproducible.
type keywordDifficultyResult struct {
	AverageAppScore    float64
	MinimumAppScore    float64
	NormalizedAppCount float64
	Difficulty         float64
	MinDifficulty      float64
	Fallback           bool
}

func clampUnit(value float64) float64 {
	return math.Min(math.Max(value, 0), 1)
}

// scoreCompetitorApp derives one competitor's app score from public metadata.
// Missing or unparseable dates degrade to a one-year window instead of being
// dropped, so a competitor is never scored as brand new by accident.
func scoreCompetitorApp(app competitorApp, keyword string, now time.Time) asc.KeywordScoreSignals {
	daysSinceFirstRelease := daysSince(app.ReleaseDate, now)
	daysSinceLastRelease := daysSince(app.CurrentVersionReleaseDate, now)

	normalizedRatingCount := clampUnit(float64(app.UserRatingCount) / keywordRatingCountCeiling)

	normalizedAverageRating := 0.0
	if app.AverageUserRating > keywordRatingFloor {
		confidence := math.Min(float64(app.UserRatingCount), keywordRatingConfidenceMin) / keywordRatingConfidenceMin
		normalizedAverageRating = clampUnit((app.AverageUserRating-keywordRatingFloor)/2) * confidence
	}

	normalizedAge := 1 - clampUnit(daysSinceLastRelease/keywordAgeWindowDays)

	ratingsPerDay := float64(app.UserRatingCount) / daysSinceFirstRelease
	if math.IsNaN(ratingsPerDay) || math.IsInf(ratingsPerDay, 0) {
		ratingsPerDay = 0
	}
	normalizedRatingsPerDay := normalizeRatingsPerDay(ratingsPerDay)

	match := detectKeywordMatch(keyword, app.Name, app.Subtitle)
	matchScore := keywordMatchScore(match)

	appScore := math.Max(
		0,
		keywordWeightRatingCount*normalizedRatingCount+
			keywordWeightAverageRating*normalizedAverageRating+
			keywordWeightAge*normalizedAge+
			keywordWeightKeywordMatch*matchScore+
			keywordWeightRatingsPerDay*normalizedRatingsPerDay,
	)

	return asc.KeywordScoreSignals{
		AppID:                     app.AppID,
		Name:                      app.Name,
		Subtitle:                  app.Subtitle,
		PublisherName:             app.PublisherName,
		AverageUserRating:         app.AverageUserRating,
		UserRatingCount:           app.UserRatingCount,
		ReleaseDate:               app.ReleaseDate,
		CurrentVersionReleaseDate: app.CurrentVersionReleaseDate,
		DaysSinceFirstRelease:     daysSinceFirstRelease,
		DaysSinceLastRelease:      daysSinceLastRelease,
		NormalizedRatingCount:     normalizedRatingCount,
		NormalizedAverageRating:   normalizedAverageRating,
		NormalizedAge:             normalizedAge,
		RatingsPerDay:             ratingsPerDay,
		NormalizedRatingsPerDay:   normalizedRatingsPerDay,
		KeywordMatch:              match,
		KeywordMatchScore:         matchScore,
		AppScore:                  appScore,
	}
}

// normalizeRatingsPerDay compresses review velocity: the first rating per day
// is worth a quarter of the signal and the remainder scales to a ceiling of
// 100 ratings per day.
func normalizeRatingsPerDay(ratingsPerDay float64) float64 {
	switch {
	case ratingsPerDay <= 0:
		return 0
	case ratingsPerDay <= 1:
		return ratingsPerDay * 0.25
	case ratingsPerDay < keywordRatingsPerDayCeil:
		return 0.25 + 0.75*((ratingsPerDay-1)/(keywordRatingsPerDayCeil-1))
	default:
		return 1
	}
}

// daysSince returns whole-precision days since an ISO-8601 date, floored at
// one day. Missing or unparseable dates degrade to a one-year window.
func daysSince(value string, now time.Time) float64 {
	parsed, ok := parsePublicDate(value)
	if !ok {
		return keywordMissingDateDays
	}
	days := now.Sub(parsed).Hours() / 24
	if math.IsNaN(days) || math.IsInf(days, 0) {
		return keywordMissingDateDays
	}
	return math.Max(days, 1)
}

func parsePublicDate(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z0700", "2006-01-02"} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

// keywordMatchScore maps a ladder rung to its weight.
func keywordMatchScore(match string) float64 {
	switch match {
	case keywordMatchTitleExactPhrase:
		return 1.0
	case keywordMatchTitleAllWords:
		return 0.8
	case keywordMatchSubtitleExactPhrase:
		return 0.5
	case keywordMatchCombinedPhrase, keywordMatchSubtitleAllWords:
		return 0.4
	default:
		return 0
	}
}

// detectKeywordMatch walks the match ladder in strength order and returns the
// first rung that holds.
func detectKeywordMatch(keyword, title, subtitle string) string {
	phrase := normalizeKeywordText(keyword)
	if phrase == "" {
		return keywordMatchNone
	}
	normalizedTitle := normalizeKeywordText(title)
	normalizedSubtitle := normalizeKeywordText(subtitle)
	phraseTokens := strings.Fields(phrase)

	switch {
	case containsTokenPhrase(normalizedTitle, phrase):
		return keywordMatchTitleExactPhrase
	case containsAllTokens(normalizedTitle, phraseTokens):
		return keywordMatchTitleAllWords
	case containsTokenPhrase(normalizedSubtitle, phrase):
		return keywordMatchSubtitleExactPhrase
	case containsTokenPhrase(strings.TrimSpace(normalizedTitle+" "+normalizedSubtitle), phrase):
		return keywordMatchCombinedPhrase
	case containsAllTokens(normalizedSubtitle, phraseTokens):
		return keywordMatchSubtitleAllWords
	default:
		return keywordMatchNone
	}
}

// containsTokenPhrase matches a contiguous sequence of complete normalized
// tokens, never a substring inside a larger token.
func containsTokenPhrase(value, phrase string) bool {
	if value == "" || phrase == "" {
		return false
	}
	return strings.Contains(" "+value+" ", " "+phrase+" ")
}

func containsAllTokens(value string, tokens []string) bool {
	if value == "" || len(tokens) == 0 {
		return false
	}
	present := make(map[string]struct{})
	for _, token := range strings.Fields(value) {
		present[token] = struct{}{}
	}
	for _, token := range tokens {
		if _, ok := present[token]; !ok {
			return false
		}
	}
	return true
}

// normalizeKeywordText applies NFKC normalization, replaces every
// non-letter/number/mark rune with a space, lowercases, and collapses runs of
// whitespace. NFKC folds compatibility forms and unifies the encodings of one
// character, but it never strips marks, so accents remain significant.
func normalizeKeywordText(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range norm.NFKC.String(value) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) {
			builder.WriteRune(unicode.ToLower(r))
			continue
		}
		builder.WriteRune(' ')
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

// computeKeywordDifficulty aggregates competitor app scores into a keyword
// difficulty. The minimum app score is weighted most heavily because the
// weakest app in the top results is the realistic entry point.
func computeKeywordDifficulty(appScores []float64, appCount int) keywordDifficultyResult {
	total := 0.0
	minimum := math.Inf(1)
	for _, score := range appScores {
		total += score
		minimum = math.Min(minimum, score)
	}
	average := 0.0
	if len(appScores) > 0 {
		average = total / float64(len(appScores))
	}
	if math.IsInf(minimum, 1) {
		minimum = 0
	}

	normalizedAppCount := 0.0
	switch {
	case appCount <= keywordAppCountFloor:
		normalizedAppCount = 0
	case appCount >= keywordAppCountCeil:
		normalizedAppCount = 1
	default:
		normalizedAppCount = float64(appCount-keywordAppCountFloor) / float64(keywordAppCountCeil-keywordAppCountFloor)
	}
	if appCount < keywordFallbackApps {
		return keywordDifficultyResult{
			AverageAppScore:    average,
			MinimumAppScore:    minimum,
			NormalizedAppCount: normalizedAppCount,
			Difficulty:         1,
			MinDifficulty:      1,
			Fallback:           true,
		}
	}

	difficulty := 100 * (0.5*normalizedAppCount + 2*average + 4*minimum) / keywordDifficultyNorm
	return keywordDifficultyResult{
		AverageAppScore:    average,
		MinimumAppScore:    minimum,
		NormalizedAppCount: normalizedAppCount,
		Difficulty:         math.Min(math.Max(difficulty, 1), 100),
		MinDifficulty:      100 * minimum,
	}
}

// isBrandKeyword reports whether a keyword most likely names the publisher of
// the leading result. It first requires the leader's publisher name to cover
// every keyword token, then accepts either a leader with real scale or a
// result set whose independent publishers are large enough that the leader's
// position is best explained by the brand rather than by the keyword.
func isBrandKeyword(keyword string, apps []competitorApp) bool {
	if len(apps) == 0 {
		return false
	}
	keywordTokens := keywordBrandTokenPattern.FindAllString(strings.ToLower(keyword), -1)
	if len(keywordTokens) == 0 {
		return false
	}

	leader := apps[0]
	publisherTokens := make(map[string]struct{})
	for _, token := range keywordBrandTokenPattern.FindAllString(strings.ToLower(leader.PublisherName), -1) {
		publisherTokens[token] = struct{}{}
	}
	for _, token := range keywordTokens {
		if _, ok := publisherTokens[token]; !ok {
			return false
		}
	}
	if leader.UserRatingCount >= keywordBrandLeaderRatings {
		return true
	}

	peerRatings := make([]float64, 0, len(apps))
	for _, app := range apps[1:] {
		if strings.EqualFold(strings.TrimSpace(app.PublisherName), strings.TrimSpace(leader.PublisherName)) {
			continue
		}
		peerRatings = append(peerRatings, float64(app.UserRatingCount))
	}
	if len(peerRatings) == 0 {
		return false
	}
	return medianOf(peerRatings) >= keywordBrandPeerRatings
}

func medianOf(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}
