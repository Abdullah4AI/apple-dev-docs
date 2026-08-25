package asc

import (
	"strconv"
	"strings"
)

// KeywordRankRow is one keyword's position in the public App Store search
// result window. Rank is null whenever the app is absent from that window.
type KeywordRankRow struct {
	Keyword      string `json:"keyword"`
	Rank         *int   `json:"rank"`
	TotalResults *int   `json:"totalResults,omitempty"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

// KeywordRankSummary counts keyword outcomes by status.
type KeywordRankSummary struct {
	Keywords    int `json:"keywords"`
	Ranked      int `json:"ranked"`
	Absent      int `json:"absent"`
	Unavailable int `json:"unavailable"`
}

// KeywordRankReport is the stable JSON contract emitted by keyword rank
// evaluation.
type KeywordRankReport struct {
	SchemaVersion string             `json:"schemaVersion"`
	GeneratedAt   string             `json:"generatedAt,omitempty"`
	AppID         string             `json:"appId"`
	Country       string             `json:"country"`
	Platform      string             `json:"platform"`
	Workers       int                `json:"workers"`
	Summary       KeywordRankSummary `json:"summary"`
	Rows          []KeywordRankRow   `json:"rows"`
}

func keywordRankSummaryRows(report *KeywordRankReport) ([]string, [][]string) {
	if report == nil {
		report = &KeywordRankReport{}
	}
	return []string{"Field", "Value"}, [][]string{
		{"App", report.AppID},
		{"Store", report.Country + " · " + keywordRankPlatformLabel(report.Platform)},
		{"Keywords", strconv.Itoa(report.Summary.Keywords)},
		{"Ranked", strconv.Itoa(report.Summary.Ranked)},
		{"Absent", strconv.Itoa(report.Summary.Absent)},
		{"Unavailable", strconv.Itoa(report.Summary.Unavailable)},
	}
}

func keywordRankRows(report *KeywordRankReport) ([]string, [][]string) {
	headers := []string{"Keyword", "Rank", "Results", "Status", "Error"}
	if report == nil {
		return headers, nil
	}

	rows := make([][]string, 0, len(report.Rows))
	for _, row := range report.Rows {
		rows = append(rows, []string{
			row.Keyword,
			formatOptionalInt(row.Rank),
			formatOptionalInt(row.TotalResults),
			row.Status,
			compactKeywordRankError(row.Error),
		})
	}
	return headers, rows
}

func keywordRankPlatformLabel(platform string) string {
	switch strings.ToUpper(strings.TrimSpace(platform)) {
	case "IOS":
		return "iOS"
	case "TV_OS", "TVOS":
		return "tvOS"
	default:
		return strings.TrimSpace(platform)
	}
}

func compactKeywordRankError(value string) string {
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
