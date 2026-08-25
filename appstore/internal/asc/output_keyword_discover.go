package asc

import "strconv"

// KeywordSuggestion is one term returned by the official advertising
// suggestion endpoints, carrying the endpoint category it came from.
type KeywordSuggestion struct {
	Keyword    string `json:"keyword"`
	Source     string `json:"source"`
	Popularity *int   `json:"popularity,omitempty"`
}

// KeywordDiscoverSourceStatus describes the availability of one discovery
// input.
type KeywordDiscoverSourceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Count  int    `json:"count"`
	Scope  string `json:"scope,omitempty"`
	Error  string `json:"error,omitempty"`
}

// KeywordDiscoverSummary counts suggestion coverage.
type KeywordDiscoverSummary struct {
	Suggestions   int `json:"suggestions"`
	Available     int `json:"available"`
	KeywordSource int `json:"keywordSource"`
	PhraseSource  int `json:"phraseSource"`
	Duplicates    int `json:"duplicates"`
	ScoreReady    int `json:"scoreReady"`
}

// KeywordDiscoverReport is the stable output contract emitted by keyword
// discovery.
type KeywordDiscoverReport struct {
	SchemaVersion string                        `json:"schemaVersion"`
	GeneratedAt   string                        `json:"generatedAt,omitempty"`
	AppID         string                        `json:"appId"`
	Country       string                        `json:"country"`
	Genre         string                        `json:"genre,omitempty"`
	Limit         int                           `json:"limit"`
	Truncated     bool                          `json:"truncated"`
	Sources       []KeywordDiscoverSourceStatus `json:"sources"`
	Summary       KeywordDiscoverSummary        `json:"summary"`
	ScoreKeywords string                        `json:"scoreKeywords"`
	Keywords      []KeywordSuggestion           `json:"keywords"`
}

func keywordDiscoverTables(report *KeywordDiscoverReport, render func([]string, [][]string)) error {
	summaryRows := [][]string{
		{"App", report.AppID},
		{"Store", report.Country},
		{"Suggestions", strconv.Itoa(report.Summary.Suggestions) + " of " + strconv.Itoa(report.Summary.Available)},
		{"From Keywords", strconv.Itoa(report.Summary.KeywordSource)},
		{"From Phrases", strconv.Itoa(report.Summary.PhraseSource)},
		{"Duplicates Removed", strconv.Itoa(report.Summary.Duplicates)},
		{"Ready To Score", strconv.Itoa(report.Summary.ScoreReady)},
	}
	if report.Genre != "" {
		summaryRows = append(summaryRows, []string{"Genre", formatKeywordScoreSourceName(report.Genre)})
	}
	if report.Truncated {
		summaryRows = append(summaryRows, []string{"Truncated", "true (--limit " + strconv.Itoa(report.Limit) + ")"})
	}
	render([]string{"Field", "Value"}, summaryRows)

	rows := make([][]string, 0, len(report.Keywords))
	for _, suggestion := range report.Keywords {
		rows = append(rows, []string{
			suggestion.Keyword,
			suggestion.Source,
			formatKeywordScoreInt(suggestion.Popularity),
		})
	}
	render([]string{"Keyword", "Source", "Popularity"}, rows)

	sourceRows := make([][]string, 0, len(report.Sources))
	for _, source := range report.Sources {
		sourceRows = append(sourceRows, []string{
			formatKeywordScoreSourceName(source.Name),
			source.Status,
			strconv.Itoa(source.Count),
			source.Scope,
			compactKeywordScoreDiagnostic(source.Error),
		})
	}
	render([]string{"Sources", "Status", "Count", "Scope", "Notes"}, sourceRows)

	if report.ScoreKeywords != "" {
		render([]string{"Score Input", "Value"}, [][]string{{"--keywords", report.ScoreKeywords}})
	}
	return nil
}
