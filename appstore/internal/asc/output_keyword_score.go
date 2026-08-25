package asc

import (
	"strconv"
	"strings"
)

// KeywordPopularity is the official search-demand snapshot for one term.
type KeywordPopularity struct {
	Country           string `json:"country,omitempty"`
	Genre             string `json:"genre,omitempty"`
	Week              string `json:"week,omitempty"`
	RankInGenre       *int   `json:"rankInGenre,omitempty"`
	PopularityInGenre *int   `json:"popularityInGenre,omitempty"`
	Popularity100     *int   `json:"popularity100,omitempty"`
	Popularity5       *int   `json:"popularity5,omitempty"`
}

// KeywordScoreSignals is the named raw evidence behind one competitor's app
// score. Date fields remain present when unavailable so degraded inputs are
// explicit and the score can still be re-derived from the JSON response.
type KeywordScoreSignals struct {
	AppID                     string  `json:"appId"`
	Name                      string  `json:"name"`
	Subtitle                  string  `json:"subtitle"`
	PublisherName             string  `json:"publisherName"`
	AverageUserRating         float64 `json:"averageUserRating"`
	UserRatingCount           int64   `json:"userRatingCount"`
	ReleaseDate               string  `json:"releaseDate"`
	CurrentVersionReleaseDate string  `json:"currentVersionReleaseDate"`
	DaysSinceFirstRelease     float64 `json:"daysSinceFirstRelease"`
	DaysSinceLastRelease      float64 `json:"daysSinceLastRelease"`
	NormalizedRatingCount     float64 `json:"normalizedRatingCount"`
	NormalizedAverageRating   float64 `json:"normalizedAverageRating"`
	NormalizedAge             float64 `json:"normalizedAge"`
	RatingsPerDay             float64 `json:"ratingsPerDay"`
	NormalizedRatingsPerDay   float64 `json:"normalizedRatingsPerDay"`
	KeywordMatch              string  `json:"keywordMatch"`
	KeywordMatchScore         float64 `json:"keywordMatchScore"`
	AppScore                  float64 `json:"appScore"`
}

// KeywordScoreRow is one evaluated keyword. Computed fields are null whenever
// the source they depend on was unavailable, so an absent score is never
// reported as a zero score.
type KeywordScoreRow struct {
	Keyword            string                `json:"keyword"`
	Status             string                `json:"status"`
	Popularity         *KeywordPopularity    `json:"popularity"`
	DifficultyScore    *float64              `json:"difficultyScore"`
	MinDifficultyScore *float64              `json:"minDifficultyScore"`
	IsBrandKeyword     *bool                 `json:"isBrandKeyword"`
	AppCount           *int                  `json:"appCount"`
	KeywordMatch       string                `json:"keywordMatch,omitempty"`
	Rank               *int                  `json:"rank,omitempty"`
	Fallback           bool                  `json:"fallback,omitempty"`
	AverageAppScore    *float64              `json:"averageAppScore,omitempty"`
	MinimumAppScore    *float64              `json:"minimumAppScore,omitempty"`
	NormalizedAppCount *float64              `json:"normalizedAppCount,omitempty"`
	RawSignals         []KeywordScoreSignals `json:"rawSignals,omitempty"`
	Error              string                `json:"error,omitempty"`
}

// KeywordScoreSourceStatus describes the availability of one score input.
type KeywordScoreSourceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Count  int    `json:"count"`
	Error  string `json:"error,omitempty"`
}

// KeywordScoreSummary counts keyword outcomes by status.
type KeywordScoreSummary struct {
	Keywords     int `json:"keywords"`
	Scored       int `json:"scored"`
	Unavailable  int `json:"unavailable"`
	WithRank     int `json:"withRank"`
	BrandMatches int `json:"brandMatches"`
}

// KeywordScoreReport is the stable output contract emitted by keyword scoring.
type KeywordScoreReport struct {
	SchemaVersion string                     `json:"schemaVersion"`
	GeneratedAt   string                     `json:"generatedAt,omitempty"`
	AppID         string                     `json:"appId,omitempty"`
	Country       string                     `json:"country"`
	Genre         string                     `json:"genre,omitempty"`
	Workers       int                        `json:"workers"`
	Sources       []KeywordScoreSourceStatus `json:"sources"`
	Summary       KeywordScoreSummary        `json:"summary"`
	Rows          []KeywordScoreRow          `json:"rows"`
}

func keywordScoreTables(report *KeywordScoreReport, render func([]string, [][]string)) error {
	summaryRows := [][]string{
		{"Store", report.Country},
		{"Keywords", strconv.Itoa(report.Summary.Keywords)},
		{"Scored", strconv.Itoa(report.Summary.Scored)},
		{"Unavailable", strconv.Itoa(report.Summary.Unavailable)},
		{"Brand Matches", strconv.Itoa(report.Summary.BrandMatches)},
	}
	if report.AppID != "" {
		summaryRows = append([][]string{{"App", report.AppID}}, summaryRows...)
		summaryRows = append(summaryRows, []string{"Ranked", strconv.Itoa(report.Summary.WithRank)})
	}
	if report.Genre != "" {
		summaryRows = append(summaryRows, []string{"Genre", formatKeywordScoreSourceName(report.Genre)})
	}
	render([]string{"Field", "Value"}, summaryRows)

	rows := make([][]string, 0, len(report.Rows))
	for _, row := range report.Rows {
		rows = append(rows, []string{
			row.Keyword,
			formatKeywordScoreFloat(row.DifficultyScore),
			formatKeywordScoreFloat(row.MinDifficultyScore),
			strconv.FormatBool(row.Fallback),
			formatKeywordScorePopularity(row.Popularity),
			formatKeywordScoreInt(row.AppCount),
			formatKeywordScoreInt(row.Rank),
			formatKeywordScoreMatch(row.KeywordMatch),
			formatKeywordScoreBool(row.IsBrandKeyword),
			row.Status,
			compactKeywordScoreDiagnostic(row.Error),
		})
	}
	render(
		[]string{"Keyword", "Difficulty", "Min Difficulty", "Fallback", "Popularity", "Apps", "Rank", "Match", "Brand", "Status", "Error"},
		rows,
	)

	sourceRows := make([][]string, 0, len(report.Sources))
	for _, source := range report.Sources {
		sourceRows = append(sourceRows, []string{
			formatKeywordScoreSourceName(source.Name),
			source.Status,
			strconv.Itoa(source.Count),
			compactKeywordScoreDiagnostic(source.Error),
		})
	}
	render([]string{"Sources", "Status", "Count", "Notes"}, sourceRows)
	return nil
}

func formatKeywordScoreFloat(value *float64) string {
	if value == nil {
		return "—"
	}
	return strconv.FormatFloat(*value, 'f', 1, 64)
}

func formatKeywordScorePopularity(popularity *KeywordPopularity) string {
	if popularity == nil {
		return "—"
	}
	return formatKeywordScoreInt(popularity.Popularity5) + " / " + formatKeywordScoreInt(popularity.Popularity100)
}

func formatKeywordScoreInt(value *int) string {
	if value == nil {
		return "—"
	}
	return strconv.Itoa(*value)
}

func formatKeywordScoreBool(value *bool) string {
	if value == nil {
		return "—"
	}
	return strconv.FormatBool(*value)
}

func formatKeywordScoreMatch(match string) string {
	switch match {
	case "":
		return "—"
	case "none":
		return "none"
	default:
		words := make([]string, 0, 3)
		var current strings.Builder
		for _, r := range match {
			if r >= 'A' && r <= 'Z' && current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			current.WriteRune(r)
		}
		if current.Len() > 0 {
			words = append(words, current.String())
		}
		return strings.ToLower(strings.Join(words, " "))
	}
}

func formatKeywordScoreSourceName(value string) string {
	words := strings.Fields(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", " "))
	for index, word := range words {
		if word == "cpa" {
			words[index] = "CPA"
			continue
		}
		words[index] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func compactKeywordScoreDiagnostic(value string) string {
	compact := strings.TrimSpace(value)
	lower := strings.ToLower(compact)
	htmlIndex := len(compact)
	for _, marker := range []string{"<!doctype html", "<html", "<head", "<body"} {
		if index := strings.Index(lower, marker); index >= 0 && index < htmlIndex {
			htmlIndex = index
		}
	}
	if htmlIndex < len(compact) {
		compact = strings.TrimSpace(compact[:htmlIndex])
	}
	compact = strings.TrimSpace(strings.TrimSuffix(compact, ":"))
	compact = strings.Join(strings.Fields(compact), " ")
	const maxRunes = 72
	runes := []rune(compact)
	if len(runes) > maxRunes {
		compact = string(runes[:maxRunes-1]) + "…"
	}
	return compact
}
