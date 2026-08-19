package optimize

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/ads"
)

func TestBuildSearchPlanJoinsOfficialEvidenceWithoutInventingMissingValues(t *testing.T) {
	input := searchPlanBuildInput{
		AppID:     "123456789",
		Version:   "4.4.4",
		VersionID: "version-1",
		Platform:  "IOS",
		Country:   "US",
		Genre:     "PRODUCTIVITY_UTILITIES",
		Locale:    "en-US",
		Window: searchPlanWindow{
			Start: "2026-07-19",
			End:   "2026-08-17",
		},
		Metadata: searchMetadataSnapshot{
			Name:     "Focus Keeper",
			Subtitle: "A simple habit tracker",
			Keywords: "focus,timer",
		},
		Ads: ads.SearchOptimizationData{
			TargetCPASuggestion:            json.RawMessage(`{"suggestedTargetCPA":{"amount":"1.20","currency":"USD"}}`),
			DailyBudgetRecommendationItems: []json.RawMessage{json.RawMessage(`{"id":"budget-1"}`)},
			DailyBudgetRecommendations:     1,
			Sources: []ads.SearchOptimizationSourceStatus{
				{Name: "keyword_suggestions", Status: "available", Count: 2},
				{Name: "search_term_popularity", Status: "available", Count: 3},
				{Name: "impression_share", Status: "available", Count: 2},
				{Name: "search_term_performance", Status: "available", Count: 3},
				{Name: "targeting_keywords", Status: "available", Count: 1},
				{Name: "negative_keywords", Status: "empty"},
			},
			Suggestions: []ads.SearchSuggestion{
				{Text: "daily habits", Popularity: intPtr(72), Kind: "keyword"},
				{Text: "mood journal", Popularity: intPtr(69), Kind: "phrase"},
			},
			Popularities: []ads.SearchPopularity{
				{Term: "habit tracker", Popularity100: intPtr(88), Popularity5: intPtr(5), RankInGenre: intPtr(1)},
				{Term: "daily habits", Popularity100: intPtr(72), RankInGenre: intPtr(8)},
				{Term: "free planner", Popularity100: intPtr(64), RankInGenre: intPtr(15)},
			},
			ImpressionShares: []ads.SearchImpressionShare{
				{Term: "habit tracker", Low: floatPtr(0.07), High: floatPtr(0.07), Rank: intPtr(6), Popularity5: intPtr(4)},
				{Term: "mood journal", Low: floatPtr(0.91), High: floatPtr(1), Rank: intPtr(1)},
			},
			SearchTerms: []ads.SearchTermPerformance{
				{Term: "habit tracker", KeywordText: "habits", MatchType: "BROAD", CampaignID: 44, AdGroupID: 55, Impressions: 1000, Taps: 70, TotalInstalls: 31, SpendAmount: "36.58", SpendCurrency: "USD"},
				{Term: "free planner", KeywordText: "planner", MatchType: "BROAD", CampaignID: 44, AdGroupID: 55, Impressions: 500, Taps: 12, TotalInstalls: 0, SpendAmount: "8.40", SpendCurrency: "USD"},
				{Term: "mood journal", KeywordText: "mood journal", MatchType: "EXACT", CampaignID: 44, AdGroupID: 55, Impressions: 800, Taps: 50, TotalInstalls: 18, SpendAmount: "18.00", SpendCurrency: "USD"},
			},
			Keywords: []ads.SearchTargetingKeyword{
				{Text: "mood journal", MatchType: "EXACT", CampaignID: 44, AdGroupID: 55},
			},
		},
	}

	report := buildSearchPlan(input)
	if report.SchemaVersion != "1" {
		t.Fatalf("SchemaVersion = %q, want 1", report.SchemaVersion)
	}
	if len(report.Recommendations.DailyBudgets) != 1 || !strings.Contains(string(report.Recommendations.DailyBudgets[0]), "budget-1") {
		t.Fatalf("recommendations = %+v", report.Recommendations)
	}
	if !strings.Contains(string(report.Recommendations.TargetCPASuggestion), `"amount":"1.20"`) {
		t.Fatalf("target CPA suggestion = %s", report.Recommendations.TargetCPASuggestion)
	}

	habit := findSearchPlanRow(t, report.Rows, "habit tracker")
	if habit.Popularity100 == nil || *habit.Popularity100 != 88 {
		t.Fatalf("habit popularity = %v, want 88", habit.Popularity100)
	}
	if habit.Popularity5 == nil || *habit.Popularity5 != 5 {
		t.Fatalf("habit 1-to-5 popularity = %v, want 5", habit.Popularity5)
	}
	if habit.ImpressionSharePopularity5 == nil || *habit.ImpressionSharePopularity5 != 4 {
		t.Fatalf("habit impression-share popularity = %v, want 4", habit.ImpressionSharePopularity5)
	}
	if habit.CPA == nil || habit.CPA.Amount != "1.18" || habit.CPA.Currency != "USD" {
		t.Fatalf("habit CPA = %+v, want USD 1.18", habit.CPA)
	}
	if !slices.Contains(habit.MetadataFields, "subtitle") {
		t.Fatalf("habit metadata fields = %v, want subtitle", habit.MetadataFields)
	}
	assertActions(t, habit.Actions, "promote_exact", "defend")
	if slices.Contains(habit.Actions, "metadata_candidate") {
		t.Fatalf("habit actions = %v, must not duplicate covered metadata", habit.Actions)
	}
	if habit.Confidence != "proven" {
		t.Fatalf("habit confidence = %q, want proven", habit.Confidence)
	}

	daily := findSearchPlanRow(t, report.Rows, "daily habits")
	assertActions(t, daily.Actions, "metadata_candidate", "untested_candidate")
	if daily.TotalInstalls != nil {
		t.Fatalf("daily habits installs = %v, want unavailable rather than zero", daily.TotalInstalls)
	}
	if daily.SuggestionPopularity == nil || *daily.SuggestionPopularity != 72 {
		t.Fatalf("daily suggestion popularity = %v, want 72", daily.SuggestionPopularity)
	}

	negative := findSearchPlanRow(t, report.Rows, "free planner")
	assertActions(t, negative.Actions, "negative_candidate")
	if negative.TotalInstalls == nil || *negative.TotalInstalls != 0 {
		t.Fatalf("free planner installs = %v, want observed zero", negative.TotalInstalls)
	}

	saturated := findSearchPlanRow(t, report.Rows, "mood journal")
	assertActions(t, saturated.Actions, "saturated", "metadata_candidate")
	if slices.Contains(saturated.Actions, "promote_exact") {
		t.Fatalf("mood actions = %v, existing exact target must suppress promotion", saturated.Actions)
	}
}

func TestBuildSearchPlanSuppressesExistingNegativeCandidate(t *testing.T) {
	report := buildSearchPlan(searchPlanBuildInput{
		Metadata: searchMetadataSnapshot{},
		Ads: ads.SearchOptimizationData{
			Sources:          []ads.SearchOptimizationSourceStatus{{Name: "negative_keywords", Status: "available", Count: 1}},
			SearchTerms:      []ads.SearchTermPerformance{{Term: "free planner", CampaignID: 44, AdGroupID: 55, Taps: 20, TotalInstalls: 0}},
			NegativeKeywords: []ads.SearchNegativeKeyword{{Text: "free planner", CampaignID: 44, AdGroupID: 55, MatchType: "EXACT"}},
		},
	})
	row := findSearchPlanRow(t, report.Rows, "free planner")
	if slices.Contains(row.Actions, "negative_candidate") {
		t.Fatalf("actions = %v, existing negative must suppress candidate", row.Actions)
	}
}

func TestBuildSearchPlanSuppressesNegativeCandidateWhenSearchTermEvidenceIsIncomplete(t *testing.T) {
	report := buildSearchPlan(searchPlanBuildInput{
		Metadata: searchMetadataSnapshot{},
		Ads: ads.SearchOptimizationData{
			Sources: []ads.SearchOptimizationSourceStatus{
				{Name: "negative_keywords", Status: "empty"},
				{Name: "search_term_performance", Status: "unavailable", Error: "one campaign failed"},
			},
			SearchTerms: []ads.SearchTermPerformance{{Term: "free planner", CampaignID: 44, AdGroupID: 55, Taps: 20, TotalInstalls: 0}},
		},
	})
	row := findSearchPlanRow(t, report.Rows, "free planner")
	if slices.Contains(row.Actions, "negative_candidate") {
		t.Fatalf("actions = %v, incomplete search-term evidence must suppress negative candidate", row.Actions)
	}
}

func TestBuildSearchPlanDoesNotInferAbsenceFromUnavailableSources(t *testing.T) {
	report := buildSearchPlan(searchPlanBuildInput{
		Metadata: searchMetadataSnapshot{},
		Ads: ads.SearchOptimizationData{
			Sources: []ads.SearchOptimizationSourceStatus{
				{Name: "targeting_keywords", Status: "unavailable", Error: "denied"},
				{Name: "negative_keywords", Status: "unavailable", Error: "denied"},
				{Name: "search_term_performance", Status: "unavailable", Error: "denied"},
			},
			Suggestions: []ads.SearchSuggestion{{Text: "untested term", Kind: "keyword"}},
			SearchTerms: []ads.SearchTermPerformance{
				{Term: "converting broad", MatchType: "BROAD", CampaignID: 44, AdGroupID: 55, Taps: 20, TotalInstalls: 3},
				{Term: "waste term", MatchType: "BROAD", CampaignID: 44, AdGroupID: 55, Taps: 20, TotalInstalls: 0},
			},
		},
	})

	for _, test := range []struct {
		term   string
		action string
	}{
		{term: "converting broad", action: "promote_exact"},
		{term: "waste term", action: "negative_candidate"},
		{term: "untested term", action: "untested_candidate"},
	} {
		row := findSearchPlanRow(t, report.Rows, test.term)
		if slices.Contains(row.Actions, test.action) {
			t.Fatalf("%q actions = %v; unavailable source must suppress %q", test.term, row.Actions, test.action)
		}
	}
}

func TestBuildSearchPlanUsesLatestImpressionSharePeriod(t *testing.T) {
	report := buildSearchPlan(searchPlanBuildInput{
		Ads: ads.SearchOptimizationData{ImpressionShares: []ads.SearchImpressionShare{
			{Term: "habit tracker", Day: "2026-08-17", Low: floatPtr(0.3), High: floatPtr(0.4), Rank: intPtr(3), Popularity5: intPtr(4)},
			{Term: "habit tracker", Day: "2026-08-16", Low: floatPtr(0.8), High: floatPtr(0.9), Rank: intPtr(1), Popularity5: intPtr(5)},
		}},
	})
	row := findSearchPlanRow(t, report.Rows, "habit tracker")
	if row.ImpressionSharePeriod != "2026-08-17" || row.ImpressionShareLow == nil || *row.ImpressionShareLow != 0.3 || row.ImpressionShareRank == nil || *row.ImpressionShareRank != 3 || row.ImpressionSharePopularity5 == nil || *row.ImpressionSharePopularity5 != 4 {
		t.Fatalf("latest impression share = %+v", row)
	}
}

func TestBuildSearchPlanUsesLatestPopularityPeriodDeterministically(t *testing.T) {
	build := func(popularities []ads.SearchPopularity) SearchPlanRow {
		report := buildSearchPlan(searchPlanBuildInput{
			Ads: ads.SearchOptimizationData{Popularities: popularities},
		})
		return findSearchPlanRow(t, report.Rows, "habit tracker")
	}
	old := ads.SearchPopularity{Term: "habit tracker", Week: "2026-08-02", Popularity5: intPtr(5), Popularity100: intPtr(95), RankInGenre: intPtr(1)}
	newerWeak := ads.SearchPopularity{Term: "habit tracker", Week: "2026-08-09", Popularity5: intPtr(3), Popularity100: intPtr(70), RankInGenre: intPtr(5)}
	newerStrong := ads.SearchPopularity{Term: "habit tracker", Week: "2026-08-09", Popularity5: intPtr(4), Popularity100: intPtr(85), RankInGenre: intPtr(2)}

	for _, popularities := range [][]ads.SearchPopularity{
		{old, newerWeak, newerStrong},
		{newerStrong, newerWeak, old},
	} {
		row := build(popularities)
		if row.Popularity5 == nil || *row.Popularity5 != 4 || row.Popularity100 == nil || *row.Popularity100 != 85 || row.RankInGenre == nil || *row.RankInGenre != 2 {
			t.Fatalf("selected popularity = %+v, want latest period with stable best rank", row)
		}
	}
}

func TestBuildSearchPlanChoosesStableCampaignContextForEqualPerformance(t *testing.T) {
	build := func(performance []ads.SearchTermPerformance) SearchPlanRow {
		report := buildSearchPlan(searchPlanBuildInput{
			Ads: ads.SearchOptimizationData{SearchTerms: performance},
		})
		return findSearchPlanRow(t, report.Rows, "habit tracker")
	}
	first := ads.SearchTermPerformance{Term: "habit tracker", KeywordText: "habits", MatchType: "BROAD", CampaignID: 11, AdGroupID: 101, TotalInstalls: 3}
	second := ads.SearchTermPerformance{Term: "habit tracker", KeywordText: "tracker", MatchType: "BROAD", CampaignID: 22, AdGroupID: 202, TotalInstalls: 3}

	forward := build([]ads.SearchTermPerformance{first, second})
	reverse := build([]ads.SearchTermPerformance{second, first})
	for _, row := range []SearchPlanRow{forward, reverse} {
		if row.CampaignID == nil || *row.CampaignID != 11 || row.AdGroupID == nil || *row.AdGroupID != 101 || row.MatchedKeyword != "habits" {
			t.Fatalf("selected context = %+v, want stable lowest campaign/ad group", row)
		}
	}
}

func TestSearchPlanRowsUseImpressionSharePopularityAsOneToFiveFallback(t *testing.T) {
	rows := searchPlanRows([]SearchPlanRow{{Term: "habit tracker", ImpressionSharePopularity5: intPtr(4)}})
	if len(rows) != 1 || len(rows[0]) < 2 || rows[0][1] != "4" {
		t.Fatalf("rendered rows = %#v, want impression-share popularity fallback", rows)
	}
}

func TestWriteSearchPlanArtifactsAreReviewableAndImportCompatible(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "optimization")
	report := SearchPlanReport{
		SchemaVersion: "1",
		AppID:         "123456789",
		Version:       "4.4.4",
		Locale:        "en-US",
		Metadata:      searchMetadataSnapshot{Keywords: "focus,timer"},
		Rows: []SearchPlanRow{
			{Term: "daily habits", Actions: []string{"metadata_candidate", "untested_candidate"}, AdGroupID: int64Ptr(55)},
			{Term: "habit tracker", Actions: []string{"promote_exact"}, CampaignID: int64Ptr(44), AdGroupID: int64Ptr(55)},
			{Term: "free planner", Actions: []string{"negative_candidate"}, CampaignID: int64Ptr(44), AdGroupID: int64Ptr(55)},
		},
	}

	artifacts, err := writeSearchPlanArtifacts(dir, report)
	if err != nil {
		t.Fatalf("writeSearchPlanArtifacts() error = %v", err)
	}
	if len(artifacts) != 4 {
		t.Fatalf("artifacts = %v, want four files", artifacts)
	}

	reportData, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded SearchPlanReport
	if err := json.Unmarshal(reportData, &decoded); err != nil || decoded.SchemaVersion != "1" {
		t.Fatalf("report artifact decode = (%+v, %v)", decoded, err)
	}

	csvFile, err := os.Open(filepath.Join(dir, "metadata-candidates.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(csvFile).ReadAll()
	_ = csvFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || strings.Join(rows[0], ",") != "locale,keywords" || rows[1][0] != "en-US" {
		t.Fatalf("metadata CSV = %#v", rows)
	}
	if got := rows[1][1]; got != "focus,timer,daily habits" {
		t.Fatalf("metadata keywords = %q", got)
	}

	assertJSONArtifact(t, filepath.Join(dir, "exact-keywords.json"), `{"items":[{"correlationId":0,"data":{"adGroupId":55,"text":"habit tracker","matchType":"EXACT"}}]}`)
	assertJSONArtifact(t, filepath.Join(dir, "negative-keywords.json"), `{"items":[{"correlationId":0,"data":{"campaignId":44,"adGroupId":55,"text":"free planner","matchType":"EXACT","status":"ENABLED"}}]}`)
}

func TestMetadataCandidateArtifactHonorsKeywordLimitAndDuplicates(t *testing.T) {
	report := SearchPlanReport{
		Locale:   "en-US",
		Metadata: searchMetadataSnapshot{Keywords: strings.Repeat("a", 95)},
		Rows: []SearchPlanRow{
			{Term: "tool", Actions: []string{"metadata_candidate"}},
			{Term: "tool", Actions: []string{"metadata_candidate"}},
			{Term: "x", Actions: []string{"metadata_candidate"}},
		},
	}
	data, err := buildMetadataCandidatesCSV(report)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Repeat("a", 95) + ",tool"
	if len(rows) != 2 || rows[1][1] != want {
		t.Fatalf("metadata candidates = %#v, want %q", rows, want)
	}
}

func TestWriteSearchPlanArtifactsRejectsFileAsOutputDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := writeSearchPlanArtifacts(path, SearchPlanReport{}); err == nil {
		t.Fatal("writeSearchPlanArtifacts() succeeded with a file as --out-dir")
	}
}

func TestResolveSearchPlanWindowRejectsNonWholeOrOutOfRangeDuration(t *testing.T) {
	for _, value := range []string{"1d", "31d", "36h", "nonsense"} {
		if _, err := resolveSearchPlanWindow(value, time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)); err == nil {
			t.Fatalf("resolveSearchPlanWindow(%q) succeeded, want error", value)
		}
	}
	window, err := resolveSearchPlanWindow("30d", time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if window.Start != "2026-07-19" || window.End != "2026-08-17" || window.PopularityStart != "2026-08-09" || window.PopularityEnd != "2026-08-15" {
		t.Fatalf("window = %+v", window)
	}
}

func TestResolveSearchPlanWindowWaitsForWeeklyPopularityPublication(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		wantStart string
		wantEnd   string
	}{
		{name: "Sunday before publication", now: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC), wantStart: "2026-08-02", wantEnd: "2026-08-08"},
		{name: "Monday before publication", now: time.Date(2026, 8, 17, 6, 59, 0, 0, time.UTC), wantStart: "2026-08-02", wantEnd: "2026-08-08"},
		{name: "Monday after publication", now: time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC), wantStart: "2026-08-09", wantEnd: "2026-08-15"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			window, err := resolveSearchPlanWindow("30d", test.now)
			if err != nil {
				t.Fatal(err)
			}
			if window.PopularityStart != test.wantStart || window.PopularityEnd != test.wantEnd {
				t.Fatalf("popularity window = %s through %s, want %s through %s", window.PopularityStart, window.PopularityEnd, test.wantStart, test.wantEnd)
			}
		})
	}
}

func findSearchPlanRow(t *testing.T, rows []SearchPlanRow, term string) SearchPlanRow {
	t.Helper()
	for _, row := range rows {
		if row.Term == term {
			return row
		}
	}
	t.Fatalf("missing row %q in %+v", term, rows)
	return SearchPlanRow{}
}

func assertActions(t *testing.T, got []string, want ...string) {
	t.Helper()
	for _, action := range want {
		if !slices.Contains(got, action) {
			t.Fatalf("actions = %v, want %q", got, action)
		}
	}
}

func assertJSONArtifact(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var gotValue any
	var wantValue any
	if err := json.Unmarshal(data, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("artifact %s = %s, want %s", path, data, want)
	}
}

func intPtr(value int) *int           { return &value }
func int64Ptr(value int64) *int64     { return &value }
func floatPtr(value float64) *float64 { return &value }
