package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestKeywordRankReportJSONContract(t *testing.T) {
	totalResults := 0
	report := KeywordRankReport{
		SchemaVersion: "1",
		AppID:         "1234567890",
		Country:       "US",
		Platform:      "IOS",
		Workers:       2,
		Summary: KeywordRankSummary{
			Keywords: 1,
			Absent:   1,
		},
		Rows: []KeywordRankRow{{
			Keyword:      "focus timer",
			TotalResults: &totalResults,
			Status:       "empty",
		}},
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	output := string(encoded)
	for _, want := range []string{
		`"schemaVersion":"1"`,
		`"appId":"1234567890"`,
		`"totalResults"`,
		`"rank":null`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %s: %s", want, output)
		}
	}
}

func TestKeywordRankReportRendererContract(t *testing.T) {
	rank := 2
	totalResults := 200
	report := &KeywordRankReport{
		AppID:    "1234567890",
		Country:  "US",
		Platform: "IOS",
		Summary: KeywordRankSummary{
			Keywords:    1,
			Ranked:      1,
			Unavailable: 0,
		},
		Rows: []KeywordRankRow{{
			Keyword:      "focus timer",
			Rank:         &rank,
			TotalResults: &totalResults,
			Status:       "available",
		}},
	}

	type section struct {
		headers []string
		rows    [][]string
	}
	var sections []section
	if err := renderByRegistry(report, func(headers []string, rows [][]string) {
		sections = append(sections, section{headers: headers, rows: rows})
	}); err != nil {
		t.Fatalf("renderByRegistry() error = %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("rendered sections = %d, want 2", len(sections))
	}
	if got := sections[0].headers; len(got) != 2 || got[0] != "Field" || got[1] != "Value" {
		t.Fatalf("summary headers = %v", got)
	}
	if got := sections[0].rows; len(got) == 0 || got[0][0] != "App" || got[0][1] != report.AppID {
		t.Fatalf("summary rows = %v", got)
	}
	if got := sections[1].headers; len(got) != 5 || got[0] != "Keyword" || got[1] != "Rank" || got[3] != "Status" {
		t.Fatalf("keyword headers = %v", got)
	}
	if got := sections[1].rows; len(got) != 1 || got[0][0] != "focus timer" || got[0][1] != "2" || got[0][3] != "available" {
		t.Fatalf("keyword rows = %v", got)
	}
}
