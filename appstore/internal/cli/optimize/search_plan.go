package optimize

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/ads"
)

const searchPlanSchemaVersion = "1"

type searchPlanWindow struct {
	Start           string `json:"start"`
	End             string `json:"end"`
	PopularityStart string `json:"popularityStart"`
	PopularityEnd   string `json:"popularityEnd"`
}

type searchMetadataSnapshot struct {
	Name     string `json:"name,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
	Keywords string `json:"keywords,omitempty"`
}

// SearchPlanMoney is one currency-qualified amount derived from official Ads
// reporting data.
type SearchPlanMoney struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// SearchPlanRow is one normalized term with explicit source provenance.
type SearchPlanRow struct {
	Term                       string           `json:"term"`
	Popularity5                *int             `json:"popularity5,omitempty"`
	Popularity100              *int             `json:"popularity100,omitempty"`
	PopularityInGenre          *int             `json:"popularityInGenre,omitempty"`
	RankInGenre                *int             `json:"rankInGenre,omitempty"`
	SuggestionPopularity       *int             `json:"suggestionPopularity,omitempty"`
	ImpressionSharePeriod      string           `json:"impressionSharePeriod,omitempty"`
	ImpressionSharePopularity5 *int             `json:"impressionSharePopularity5,omitempty"`
	ImpressionShareLow         *float64         `json:"impressionShareLow,omitempty"`
	ImpressionShareHigh        *float64         `json:"impressionShareHigh,omitempty"`
	ImpressionShareRank        *int             `json:"impressionShareRank,omitempty"`
	Impressions                *int64           `json:"impressions,omitempty"`
	Taps                       *int64           `json:"taps,omitempty"`
	TapInstalls                *int64           `json:"tapInstalls,omitempty"`
	TotalInstalls              *int64           `json:"totalInstalls,omitempty"`
	Spend                      *SearchPlanMoney `json:"spend,omitempty"`
	CPA                        *SearchPlanMoney `json:"cpa,omitempty"`
	MatchedKeyword             string           `json:"matchedKeyword,omitempty"`
	MatchType                  string           `json:"matchType,omitempty"`
	CampaignID                 *int64           `json:"campaignId,omitempty"`
	AdGroupID                  *int64           `json:"adGroupId,omitempty"`
	MetadataFields             []string         `json:"metadataFields,omitempty"`
	Sources                    []string         `json:"sources"`
	Actions                    []string         `json:"actions,omitempty"`
	Confidence                 string           `json:"confidence"`
}

// SearchPlanSummary summarizes evidence and recommendation coverage.
type SearchPlanSummary struct {
	Terms                      int            `json:"terms"`
	Actions                    map[string]int `json:"actions"`
	AvailableSources           int            `json:"availableSources"`
	EmptySources               int            `json:"emptySources"`
	UnavailableSources         int            `json:"unavailableSources"`
	DailyBudgetRecommendations int            `json:"dailyBudgetRecommendations"`
	TargetCPARecommendations   int            `json:"targetCpaRecommendations"`
}

// SearchPlanRecommendations preserves Apple's actionable recommendation
// payloads without converting them into client-authored advice.
type SearchPlanRecommendations struct {
	DailyBudgets        []json.RawMessage `json:"dailyBudgets,omitempty"`
	TargetCPAs          []json.RawMessage `json:"targetCpas,omitempty"`
	TargetCPASuggestion json.RawMessage   `json:"targetCpaSuggestion,omitempty"`
}

// SearchPlanReport is the stable JSON contract emitted by the workflow.
type SearchPlanReport struct {
	SchemaVersion   string                               `json:"schemaVersion"`
	GeneratedAt     string                               `json:"generatedAt,omitempty"`
	AppID           string                               `json:"appId"`
	Version         string                               `json:"version"`
	VersionID       string                               `json:"versionId,omitempty"`
	AppInfoID       string                               `json:"appInfoId,omitempty"`
	Platform        string                               `json:"platform"`
	Country         string                               `json:"country"`
	Genre           string                               `json:"genre"`
	Locale          string                               `json:"locale"`
	Window          searchPlanWindow                     `json:"window"`
	Metadata        searchMetadataSnapshot               `json:"metadata"`
	Sources         []ads.SearchOptimizationSourceStatus `json:"sources"`
	Eligibility     []ads.SearchEligibility              `json:"eligibility,omitempty"`
	Campaigns       []ads.SearchCampaign                 `json:"campaigns,omitempty"`
	Recommendations SearchPlanRecommendations            `json:"recommendations"`
	Summary         SearchPlanSummary                    `json:"summary"`
	Rows            []SearchPlanRow                      `json:"rows"`
	Notices         []string                             `json:"notices,omitempty"`
	Artifacts       []string                             `json:"artifacts,omitempty"`
}

type searchPlanBuildInput struct {
	GeneratedAt string
	AppID       string
	Version     string
	VersionID   string
	AppInfoID   string
	Platform    string
	Country     string
	Genre       string
	Locale      string
	Window      searchPlanWindow
	Metadata    searchMetadataSnapshot
	Ads         ads.SearchOptimizationData
}

type searchPlanAccumulator struct {
	row                 SearchPlanRow
	hasSuggestion       bool
	hasPerformance      bool
	hasExistingExact    bool
	hasExistingNegative bool
	bestInstalls        int64
	hasPopularity       bool
	popularityPeriod    string
	spendValue          float64
	spendCurrency       string
	spendValid          bool
	spendMixedCurrency  bool
}

type searchPlanEvidence struct {
	targetingKeywordsComplete bool
	negativeKeywordsComplete  bool
	searchTermsComplete       bool
}

func buildSearchPlan(input searchPlanBuildInput) SearchPlanReport {
	evidence := searchPlanEvidence{
		targetingKeywordsComplete: searchPlanSourceComplete(input.Ads.Sources, "targeting_keywords"),
		negativeKeywordsComplete:  searchPlanSourceComplete(input.Ads.Sources, "negative_keywords"),
		searchTermsComplete:       searchPlanSourceComplete(input.Ads.Sources, "search_term_performance"),
	}
	terms := make(map[string]*searchPlanAccumulator)
	get := func(term string) *searchPlanAccumulator {
		key := normalizeSearchTerm(term)
		if key == "" {
			return nil
		}
		if existing := terms[key]; existing != nil {
			return existing
		}
		entry := &searchPlanAccumulator{row: SearchPlanRow{Term: strings.TrimSpace(term)}}
		terms[key] = entry
		return entry
	}

	for _, suggestion := range input.Ads.Suggestions {
		entry := get(suggestion.Text)
		if entry == nil {
			continue
		}
		entry.hasSuggestion = true
		entry.row.Sources = appendUnique(entry.row.Sources, suggestion.Kind+"_suggestion")
		if greaterIntPointer(suggestion.Popularity, entry.row.SuggestionPopularity) {
			entry.row.SuggestionPopularity = copyIntPointer(suggestion.Popularity)
		}
	}
	for _, popularity := range input.Ads.Popularities {
		entry := get(popularity.Term)
		if entry == nil {
			continue
		}
		entry.row.Sources = appendUnique(entry.row.Sources, "search_term_popularity")
		if !shouldSelectSearchPlanPopularity(entry, popularity) {
			continue
		}
		entry.hasPopularity = true
		entry.popularityPeriod = searchPlanPopularityPeriod(popularity)
		entry.row.Popularity5 = copyIntPointer(popularity.Popularity5)
		entry.row.Popularity100 = copyIntPointer(popularity.Popularity100)
		entry.row.PopularityInGenre = copyIntPointer(popularity.PopularityInGenre)
		entry.row.RankInGenre = copyIntPointer(popularity.RankInGenre)
	}
	for _, share := range input.Ads.ImpressionShares {
		entry := get(share.Term)
		if entry == nil {
			continue
		}
		entry.row.Sources = appendUnique(entry.row.Sources, "impression_share")
		period := strings.TrimSpace(share.Day)
		if period == "" {
			period = strings.TrimSpace(share.Week)
		}
		if entry.row.ImpressionSharePeriod == "" || period >= entry.row.ImpressionSharePeriod {
			entry.row.ImpressionSharePeriod = period
			entry.row.ImpressionShareLow = copyFloatPointer(share.Low)
			entry.row.ImpressionShareHigh = copyFloatPointer(share.High)
			entry.row.ImpressionShareRank = copyIntPointer(share.Rank)
			entry.row.ImpressionSharePopularity5 = copyIntPointer(share.Popularity5)
		}
	}

	for _, keyword := range input.Ads.Keywords {
		if strings.EqualFold(keyword.MatchType, "EXACT") {
			if entry := get(keyword.Text); entry != nil {
				entry.hasExistingExact = true
				entry.row.Sources = appendUnique(entry.row.Sources, "targeting_keyword")
			}
		}
	}
	for _, negative := range input.Ads.NegativeKeywords {
		if entry := get(negative.Text); entry != nil {
			entry.hasExistingNegative = true
			entry.row.Sources = appendUnique(entry.row.Sources, "negative_keyword")
		}
	}
	for _, performance := range input.Ads.SearchTerms {
		entry := get(performance.Term)
		if entry == nil {
			continue
		}
		entry.hasPerformance = true
		entry.row.Sources = appendUnique(entry.row.Sources, "search_term_performance")
		addInt64Pointer(&entry.row.Impressions, performance.Impressions)
		addInt64Pointer(&entry.row.Taps, performance.Taps)
		addInt64Pointer(&entry.row.TapInstalls, performance.TapInstalls)
		addInt64Pointer(&entry.row.TotalInstalls, performance.TotalInstalls)

		if shouldSelectSearchPlanContext(entry, performance) {
			entry.bestInstalls = performance.TotalInstalls
			entry.row.MatchedKeyword = strings.TrimSpace(performance.KeywordText)
			entry.row.MatchType = strings.ToUpper(strings.TrimSpace(performance.MatchType))
			entry.row.CampaignID = int64PointerIfPositive(performance.CampaignID)
			entry.row.AdGroupID = int64PointerIfPositive(performance.AdGroupID)
		}
		if amount, err := strconv.ParseFloat(strings.TrimSpace(performance.SpendAmount), 64); err == nil && amount >= 0 && !entry.spendMixedCurrency {
			currency := strings.ToUpper(strings.TrimSpace(performance.SpendCurrency))
			switch {
			case !entry.spendValid:
				entry.spendValid = true
				entry.spendCurrency = currency
				entry.spendValue = amount
			case entry.spendCurrency == currency:
				entry.spendValue += amount
			default:
				entry.spendValid = false
				entry.spendMixedCurrency = true
				entry.spendValue = 0
				entry.spendCurrency = ""
			}
		}
	}

	rows := make([]SearchPlanRow, 0, len(terms))
	for _, entry := range terms {
		entry.row.MetadataFields = metadataCoverage(input.Metadata, entry.row.Term)
		if entry.spendValid {
			entry.row.Spend = &SearchPlanMoney{Amount: formatMoney(entry.spendValue), Currency: entry.spendCurrency}
			if entry.row.TotalInstalls != nil && *entry.row.TotalInstalls > 0 {
				entry.row.CPA = &SearchPlanMoney{Amount: formatMoney(entry.spendValue / float64(*entry.row.TotalInstalls)), Currency: entry.spendCurrency}
			}
		}
		classifySearchPlanRow(entry, evidence)
		sort.Strings(entry.row.Sources)
		rows = append(rows, entry.row)
	}
	sortSearchPlanRows(rows)

	report := SearchPlanReport{
		SchemaVersion: searchPlanSchemaVersion,
		GeneratedAt:   input.GeneratedAt,
		AppID:         input.AppID,
		Version:       input.Version,
		VersionID:     input.VersionID,
		AppInfoID:     input.AppInfoID,
		Platform:      input.Platform,
		Country:       input.Country,
		Genre:         input.Genre,
		Locale:        input.Locale,
		Window:        input.Window,
		Metadata:      input.Metadata,
		Sources:       append([]ads.SearchOptimizationSourceStatus(nil), input.Ads.Sources...),
		Eligibility:   append([]ads.SearchEligibility(nil), input.Ads.Eligibilities...),
		Campaigns:     append([]ads.SearchCampaign(nil), input.Ads.Campaigns...),
		Recommendations: SearchPlanRecommendations{
			DailyBudgets:        append([]json.RawMessage(nil), input.Ads.DailyBudgetRecommendationItems...),
			TargetCPAs:          append([]json.RawMessage(nil), input.Ads.TargetCPARecommendationItems...),
			TargetCPASuggestion: append(json.RawMessage(nil), input.Ads.TargetCPASuggestion...),
		},
		Rows: rows,
	}
	report.Summary = summarizeSearchPlan(report.Rows, report.Sources, input.Ads)
	report.Notices = searchPlanNotices(input.Ads)
	return report
}

func shouldSelectSearchPlanPopularity(entry *searchPlanAccumulator, candidate ads.SearchPopularity) bool {
	if !entry.hasPopularity {
		return true
	}
	candidatePeriod := searchPlanPopularityPeriod(candidate)
	if candidatePeriod != entry.popularityPeriod {
		return candidatePeriod > entry.popularityPeriod
	}
	if comparison := compareOptionalInt(candidate.RankInGenre, entry.row.RankInGenre, false); comparison != 0 {
		return comparison > 0
	}
	for _, values := range [][2]*int{
		{candidate.Popularity100, entry.row.Popularity100},
		{candidate.Popularity5, entry.row.Popularity5},
		{candidate.PopularityInGenre, entry.row.PopularityInGenre},
	} {
		if comparison := compareOptionalInt(values[0], values[1], true); comparison != 0 {
			return comparison > 0
		}
	}
	return false
}

func searchPlanPopularityPeriod(popularity ads.SearchPopularity) string {
	if week := strings.TrimSpace(popularity.Week); week != "" {
		return week
	}
	return strings.TrimSpace(popularity.Month)
}

func compareOptionalInt(candidate, current *int, preferHigher bool) int {
	switch {
	case candidate != nil && current == nil:
		return 1
	case candidate == nil && current != nil:
		return -1
	case candidate == nil:
		return 0
	case *candidate == *current:
		return 0
	case preferHigher && *candidate > *current:
		return 1
	case !preferHigher && *candidate < *current:
		return 1
	default:
		return -1
	}
}

func shouldSelectSearchPlanContext(entry *searchPlanAccumulator, candidate ads.SearchTermPerformance) bool {
	if candidate.TotalInstalls != entry.bestInstalls {
		return candidate.TotalInstalls > entry.bestInstalls
	}

	currentCampaignID := pointerInt64Value(entry.row.CampaignID, 0)
	switch {
	case currentCampaignID == 0 && candidate.CampaignID > 0:
		return true
	case candidate.CampaignID == 0 && currentCampaignID > 0:
		return false
	case candidate.CampaignID != currentCampaignID:
		return candidate.CampaignID < currentCampaignID
	}

	currentAdGroupID := pointerInt64Value(entry.row.AdGroupID, 0)
	switch {
	case currentAdGroupID == 0 && candidate.AdGroupID > 0:
		return true
	case candidate.AdGroupID == 0 && currentAdGroupID > 0:
		return false
	case candidate.AdGroupID != currentAdGroupID:
		return candidate.AdGroupID < currentAdGroupID
	}

	currentKeyword := strings.ToLower(strings.TrimSpace(entry.row.MatchedKeyword))
	candidateKeyword := strings.ToLower(strings.TrimSpace(candidate.KeywordText))
	if candidateKeyword != currentKeyword {
		return currentKeyword == "" || candidateKeyword < currentKeyword
	}
	return strings.ToUpper(strings.TrimSpace(candidate.MatchType)) < strings.ToUpper(strings.TrimSpace(entry.row.MatchType))
}

func resolveSearchPlanWindow(value string, now time.Time) (searchPlanWindow, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if !strings.HasSuffix(trimmed, "d") || len(trimmed) < 2 {
		return searchPlanWindow{}, fmt.Errorf("--window must be a whole number of days from 2d through 30d")
	}
	days, err := strconv.Atoi(strings.TrimSuffix(trimmed, "d"))
	if err != nil || days < 2 || days > 30 {
		return searchPlanWindow{}, fmt.Errorf("--window must be a whole number of days from 2d through 30d")
	}

	utcNow := now.UTC()
	today := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
	end := today.AddDate(0, 0, -1)
	start := end.AddDate(0, 0, -(days - 1))

	// Popularity snapshots publish at 07:00 UTC on the Monday after each week.
	popularityEnd := end
	for popularityEnd.Weekday() != time.Saturday {
		popularityEnd = popularityEnd.AddDate(0, 0, -1)
	}
	publicationTime := popularityEnd.AddDate(0, 0, 2).Add(7 * time.Hour)
	if utcNow.Before(publicationTime) {
		popularityEnd = popularityEnd.AddDate(0, 0, -7)
	}
	popularityStart := popularityEnd.AddDate(0, 0, -6)

	return searchPlanWindow{
		Start:           start.Format(time.DateOnly),
		End:             end.Format(time.DateOnly),
		PopularityStart: popularityStart.Format(time.DateOnly),
		PopularityEnd:   popularityEnd.Format(time.DateOnly),
	}, nil
}

func classifySearchPlanRow(entry *searchPlanAccumulator, evidence searchPlanEvidence) {
	row := &entry.row
	installs := int64(0)
	if row.TotalInstalls != nil {
		installs = *row.TotalInstalls
	}
	if evidence.targetingKeywordsComplete && entry.hasPerformance && installs > 0 && strings.EqualFold(row.MatchType, "BROAD") && !entry.hasExistingExact {
		row.Actions = append(row.Actions, "promote_exact")
	}
	if evidence.searchTermsComplete && evidence.negativeKeywordsComplete && entry.hasPerformance && installs == 0 && row.Taps != nil && *row.Taps >= 10 && !entry.hasExistingNegative {
		row.Actions = append(row.Actions, "negative_candidate")
	}
	if len(row.MetadataFields) == 0 && (installs > 0 || entry.hasSuggestion) {
		row.Actions = append(row.Actions, "metadata_candidate")
	}
	if installs > 0 && row.ImpressionShareLow != nil && *row.ImpressionShareLow < 0.5 {
		row.Actions = append(row.Actions, "defend")
	}
	if row.ImpressionShareLow != nil && row.ImpressionShareHigh != nil && *row.ImpressionShareLow >= 0.91 && *row.ImpressionShareHigh >= 1 {
		row.Actions = append(row.Actions, "saturated")
	}
	if evidence.searchTermsComplete && entry.hasSuggestion && !entry.hasPerformance {
		row.Actions = append(row.Actions, "untested_candidate")
	}
	switch {
	case installs > 0:
		row.Confidence = "proven"
	case entry.hasPerformance:
		row.Confidence = "observed"
	default:
		row.Confidence = "suggested"
	}
}

func searchPlanSourceComplete(sources []ads.SearchOptimizationSourceStatus, name string) bool {
	for _, source := range sources {
		if source.Name == name {
			return source.Status == "available" || source.Status == "empty"
		}
	}
	return false
}

func summarizeSearchPlan(rows []SearchPlanRow, sources []ads.SearchOptimizationSourceStatus, data ads.SearchOptimizationData) SearchPlanSummary {
	summary := SearchPlanSummary{
		Terms:                      len(rows),
		Actions:                    map[string]int{},
		DailyBudgetRecommendations: data.DailyBudgetRecommendations,
		TargetCPARecommendations:   data.TargetCPARecommendations,
	}
	for _, row := range rows {
		for _, action := range row.Actions {
			summary.Actions[action]++
		}
	}
	for _, source := range sources {
		switch source.Status {
		case "available":
			summary.AvailableSources++
		case "empty":
			summary.EmptySources++
		case "unavailable":
			summary.UnavailableSources++
		}
	}
	return summary
}

func searchPlanNotices(data ads.SearchOptimizationData) []string {
	notices := make([]string, 0)
	for _, source := range data.Sources {
		if source.Status == "unavailable" {
			notices = append(notices, fmt.Sprintf("%s unavailable: %s", source.Name, source.Error))
		}
	}
	for _, eligibility := range data.Eligibilities {
		if strings.EqualFold(eligibility.State, "INELIGIBLE") {
			notices = append(notices, fmt.Sprintf("app is ineligible for %s in %s on %s: %s", eligibility.SupplyPlacement, eligibility.Country, eligibility.DeviceClass, strings.Join(eligibility.Reasons, ", ")))
		}
	}
	return notices
}

func sortSearchPlanRows(rows []SearchPlanRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		leftPriority := searchPlanActionPriority(rows[i].Actions)
		rightPriority := searchPlanActionPriority(rows[j].Actions)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		leftPopularity := pointerIntValue(rows[i].Popularity100, pointerIntValue(rows[i].SuggestionPopularity, -1))
		rightPopularity := pointerIntValue(rows[j].Popularity100, pointerIntValue(rows[j].SuggestionPopularity, -1))
		if leftPopularity != rightPopularity {
			return leftPopularity > rightPopularity
		}
		leftInstalls := pointerInt64Value(rows[i].TotalInstalls, -1)
		rightInstalls := pointerInt64Value(rows[j].TotalInstalls, -1)
		if leftInstalls != rightInstalls {
			return leftInstalls > rightInstalls
		}
		return cmp.Less(strings.ToLower(rows[i].Term), strings.ToLower(rows[j].Term))
	})
}

func searchPlanActionPriority(actions []string) int {
	priorities := map[string]int{
		"promote_exact":      0,
		"negative_candidate": 1,
		"defend":             2,
		"metadata_candidate": 3,
		"saturated":          4,
		"untested_candidate": 5,
	}
	priority := 99
	for _, action := range actions {
		if value, ok := priorities[action]; ok && value < priority {
			priority = value
		}
	}
	return priority
}

func metadataCoverage(metadata searchMetadataSnapshot, term string) []string {
	fields := make([]string, 0, 3)
	if normalizedPhraseContains(metadata.Name, term) {
		fields = append(fields, "name")
	}
	if normalizedPhraseContains(metadata.Subtitle, term) {
		fields = append(fields, "subtitle")
	}
	normalizedTerm := normalizeSearchTerm(term)
	for _, keyword := range strings.Split(metadata.Keywords, ",") {
		if normalizeSearchTerm(keyword) == normalizedTerm && normalizedTerm != "" {
			fields = append(fields, "keywords")
			break
		}
	}
	return fields
}

func normalizedPhraseContains(field, term string) bool {
	normalizedField := normalizeSearchTerm(field)
	normalizedTerm := normalizeSearchTerm(term)
	if normalizedField == "" || normalizedTerm == "" {
		return false
	}
	return strings.Contains(" "+normalizedField+" ", " "+normalizedTerm+" ")
}

func normalizeSearchTerm(value string) string {
	var builder strings.Builder
	space := true
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			space = false
			continue
		}
		if !space {
			builder.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func appendUnique(values []string, value string) []string {
	if value == "" || slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func addInt64Pointer(target **int64, value int64) {
	if *target == nil {
		copy := value
		*target = &copy
		return
	}
	**target += value
}

func int64PointerIfPositive(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	copy := value
	return &copy
}

func copyIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func greaterIntPointer(left, right *int) bool {
	return left != nil && (right == nil || *left > *right)
}

func pointerIntValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func pointerInt64Value(value *int64, fallback int64) int64 {
	if value == nil {
		return fallback
	}
	return *value
}

func formatMoney(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}
