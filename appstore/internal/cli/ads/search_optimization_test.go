package ads

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/appleads"
)

type searchOptimizationRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn searchOptimizationRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestFetchSearchOptimizationDataUsesOfficialEndpointsAndPreservesPartialFailures(t *testing.T) {
	var mu sync.Mutex
	requests := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s for %s, want POST", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-AP-Context"); got != "adAccountId=account-1;" {
			t.Errorf("X-AP-Context = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode %s body: %v", r.URL.Path, err)
		}
		mu.Lock()
		requests[r.URL.Path]++
		mu.Unlock()
		assertOptimizationPaginationShape(t, r.URL.Path, body)

		switch r.URL.Path {
		case "/v1/campaigns/query":
			assertFilter(t, body, "promotedObjectType", "APPSTORE_APP")
			assertFilter(t, body, "promotedObjectId", "123456789")
			writeJSON(t, w, `{"result":[{"id":44,"name":"US Search","status":"ENABLED","displayStatus":"RUNNING","promotedObjectId":"123456789","targeting":{"countryOrRegion":{"include":["US"]}}}],"pagination":{"offset":0,"pageSize":1000,"totalCount":1}}`)
		case "/v1/suggestions/keywords/query":
			assertFilter(t, body, "promotedObjectId", "123456789")
			assertFilter(t, body, "promotedObjectType", "APPSTORE_APP")
			assertFilter(t, body, "countriesOrRegions", "US")
			writeJSON(t, w, `{"result":[{"text":"habit tracker","popularity":80}],"pagination":{"offset":0,"pageSize":1000,"totalCount":1}}`)
		case "/v1/suggestions/phrases/query":
			if hasOptimizationFilter(body, "countriesOrRegions") {
				t.Errorf("phrase suggestion request unexpectedly includes countriesOrRegions: %#v", body)
			}
			assertFilter(t, body, "queryType", "SUGGESTION")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"INVALID_VALUE","message":"One or more validation errors occurred.","details":[{"code":"INVALID_FILTER","message":"Unsupported filter value","info":{"field":"filters[2].value"}}]}`))
		case "/v1/suggestions/target-cpas/query":
			assertFilter(t, body, "promotedObjectId", "123456789")
			assertFilter(t, body, "promotedObjectType", "APPSTORE_APP")
			assertFilter(t, body, "countryOrRegion", "US")
			writeJSON(t, w, `{"result":{"promotedObjectId":"123456789","suggestedTargetCPA":{"amount":"1.20","currency":"USD"},"countryOrRegion":["US"],"appCategory":"Productivity"}}`)
		case "/v1/insights/apps/search-term-popularity/query":
			assertFilter(t, body, "countryOrRegion", "US")
			assertFilter(t, body, "genre", "PRODUCTIVITY_UTILITIES")
			assertNestedValue(t, body, "timeRange", "start", "2026-08-09")
			assertNestedValue(t, body, "timeRange", "end", "2026-08-15")
			assertNestedValue(t, body, "timeRange", "timeZone", "UTC")
			assertNestedValue(t, body, "timeRange", "granularity", "WEEKLY_SUN_SAT")
			assertSorting(t, body, "rankInGenre", "sortOrder", "ASC")
			writeJSON(t, w, `{"result":{"rows":[{"week":"2026-08-09","countryOrRegion":"US","genre":"PRODUCTIVITY_UTILITIES","searchTerm":"habit tracker","rankInGenre":2,"searchPopularity1to100":88,"searchPopularity1to5":5}]},"pagination":{"offset":0,"pageSize":5000,"totalCount":1}}`)
		case "/v1/insights/apps/impression-share/query":
			assertFilter(t, body, "promotedObjectId", "123456789")
			assertFilter(t, body, "countryOrRegion", "US")
			assertNestedValue(t, body, "options", "impressionShareReportType", "ALL_SLOTS")
			assertNestedValue(t, body, "timeRange", "timeZone", "UTC")
			assertNestedValue(t, body, "timeRange", "granularity", "DAILY")
			writeJSON(t, w, `{"result":{"rows":[{"day":"2026-08-17","promotedObjectId":"123456789","countryOrRegion":"US","searchTerm":"habit tracker","lowImpressionShare":0.07,"highImpressionShare":0.07,"rank":6,"searchPopularity1to5":4}]},"pagination":{"offset":0,"pageSize":5000,"totalCount":1}}`)
		case "/v1/eligibilities/apps/query":
			assertFilter(t, body, "adamId", float64(123456789))
			writeJSON(t, w, `{"result":[{"adamId":123456789,"supplyPlacement":"APPSTORE_SEARCH_RESULTS","supplySource":"APPSTORE","state":"ELIGIBLE","countryOrRegion":"US","deviceClass":"IPHONE"}],"pagination":{"offset":0,"pageSize":1000,"totalCount":1}}`)
		case "/v1/recommendations/daily-budgets/query":
			assertFilter(t, body, "promotedObjectId", "123456789")
			assertFilter(t, body, "promotedObjectType", "APPSTORE_APP")
			assertFilter(t, body, "state", "AVAILABLE")
			writeJSON(t, w, `{"result":[{"id":"budget-1","campaignId":44,"state":"AVAILABLE"}],"pagination":{"offset":0,"pageSize":1000,"totalCount":1}}`)
		case "/v1/recommendations/target-cpas/query":
			assertFilter(t, body, "promotedObjectId", "123456789")
			assertFilter(t, body, "promotedObjectType", "APPSTORE_APP")
			assertFilter(t, body, "state", "AVAILABLE")
			writeJSON(t, w, `{"result":[],"pagination":{"offset":0,"pageSize":1000,"totalCount":0}}`)
		case "/v1/keywords/query":
			assertFilter(t, body, "campaignId", float64(44))
			writeJSON(t, w, `{"result":[{"id":88,"campaignId":44,"adGroupId":55,"text":"habits","matchType":"BROAD","status":"ENABLED"}],"pagination":{"offset":0,"pageSize":1000,"totalCount":1}}`)
		case "/v1/negative-keywords/query":
			writeJSON(t, w, `{"result":[],"pagination":{"offset":0,"pageSize":1000,"totalCount":0}}`)
		case "/v1/reports/apps/searchterms/query":
			assertFilter(t, body, "campaignId", float64(44))
			assertNestedValue(t, body, "timeRange", "start", "2026-07-19")
			assertNestedValue(t, body, "timeRange", "end", "2026-08-17")
			assertNestedValue(t, body, "timeRange", "timeZone", "ORTZ")
			assertNestedValue(t, body, "timeRange", "granularity", "DAILY")
			assertArrayContains(t, body, "fields", "totalInstalls")
			assertArrayContains(t, body, "groupBy", "countryOrRegion")
			writeJSON(t, w, `{"result":{"rows":[{"metadata":{"searchTermText":"habit tracker","keyword":{"id":88,"text":"habits","matchType":"BROAD"},"campaignId":44,"adGroupId":55,"countryOrRegion":"US"},"totalMetrics":{"localSpend":{"amount":"36.58","currency":"USD"},"impressions":1000,"taps":70,"tapInstalls":28,"totalInstalls":31}}]},"pagination":{"offset":0,"pageSize":5000,"totalCount":1}}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := appleads.NewClient(
		appleads.Credentials{AccessToken: "token", AdAccountID: "account-1"},
		appleads.WithPlatformBaseURL(server.URL+"/v1/"),
	)
	if err != nil {
		t.Fatal(err)
	}

	data, err := fetchSearchOptimizationData(context.Background(), client, SearchOptimizationRequest{
		AppID:           "123456789",
		Country:         "US",
		Genre:           "PRODUCTIVITY_UTILITIES",
		Start:           "2026-07-19",
		End:             "2026-08-17",
		PopularityStart: "2026-08-09",
		PopularityEnd:   "2026-08-15",
	})
	if err != nil {
		t.Fatalf("fetchSearchOptimizationData() error = %v", err)
	}
	if len(data.Suggestions) != 1 || data.Suggestions[0].Text != "habit tracker" {
		t.Fatalf("suggestions = %+v", data.Suggestions)
	}
	if len(data.Popularities) != 1 || data.Popularities[0].Term != "habit tracker" || data.Popularities[0].Popularity5 == nil || *data.Popularities[0].Popularity5 != 5 {
		t.Fatalf("popularities = %+v", data.Popularities)
	}
	if len(data.SearchTerms) != 1 || data.SearchTerms[0].TotalInstalls != 31 || data.SearchTerms[0].AdGroupID != 55 {
		t.Fatalf("search terms = %+v", data.SearchTerms)
	}
	if data.DailyBudgetRecommendations != 1 || data.TargetCPARecommendations != 0 {
		t.Fatalf("recommendations = (%d, %d)", data.DailyBudgetRecommendations, data.TargetCPARecommendations)
	}
	if len(data.TargetCPASuggestion) == 0 || !strings.Contains(string(data.TargetCPASuggestion), `"amount":"1.20"`) {
		t.Fatalf("target CPA suggestion = %s", data.TargetCPASuggestion)
	}
	if len(data.DailyBudgetRecommendationItems) != 1 || !strings.Contains(string(data.DailyBudgetRecommendationItems[0]), "budget-1") {
		t.Fatalf("daily budget recommendation items = %s", data.DailyBudgetRecommendationItems)
	}
	phraseSource := findOptimizationSource(t, data.Sources, "phrase_suggestions")
	if phraseSource.Status != "unavailable" || !strings.Contains(phraseSource.Error, "INVALID_FILTER") || !strings.Contains(phraseSource.Error, `"field":"filters[2].value"`) {
		t.Fatalf("phrase source = %+v", phraseSource)
	}
	if requests["/v1/negative-keywords/query"] != 2 {
		t.Fatalf("negative keyword requests = %d, want campaign and ad-group scopes", requests["/v1/negative-keywords/query"])
	}
}

func TestQueryOptimizationListPaginatesRequestBody(t *testing.T) {
	var offsets []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Pagination map[string]any `json:"pagination"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode pagination request: %v", err)
			http.Error(w, "invalid test request", http.StatusBadRequest)
			return
		}
		if _, present := body.Pagination["fetchTotalCount"]; present {
			t.Errorf("recommendation-style pagination contains unsupported fetchTotalCount: %#v", body.Pagination)
		}
		offset, _ := body.Pagination["offset"].(float64)
		offsets = append(offsets, int(offset))
		if offset == 0 {
			writeJSON(t, w, `{"result":[{"text":"one","popularity":1}],"pagination":{"offset":0,"pageSize":1,"totalCount":2}}`)
			return
		}
		writeJSON(t, w, `{"result":[{"text":"two","popularity":2}],"pagination":{"offset":1,"pageSize":1,"totalCount":2}}`)
	}))
	defer server.Close()
	client, err := appleads.NewClient(appleads.Credentials{AccessToken: "token", AdAccountID: "account-1"}, appleads.WithPlatformBaseURL(server.URL+"/v1/"))
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := appleads.PlatformEndpointByCommandPath("suggestions", "keywords", "find")
	if !ok {
		t.Fatal("missing endpoint spec")
	}
	items, err := queryOptimizationList[SearchSuggestion](context.Background(), client, spec, map[string]any{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || len(offsets) != 2 || offsets[0] != 0 || offsets[1] != 1 {
		t.Fatalf("items=%+v offsets=%v", items, offsets)
	}
}

func TestQueryOptimizationListCapsUnboundedPages(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeJSON(t, w, `{"result":[{"text":"one","popularity":1}],"pagination":{"offset":0,"pageSize":1}}`)
	}))
	defer server.Close()
	client, err := appleads.NewClient(appleads.Credentials{AccessToken: "token", AdAccountID: "account-1"}, appleads.WithPlatformBaseURL(server.URL+"/v1/"))
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := appleads.PlatformEndpointByCommandPath("suggestions", "keywords", "find")
	if !ok {
		t.Fatal("missing endpoint spec")
	}

	items, err := queryOptimizationList[SearchSuggestion](context.Background(), client, spec, map[string]any{}, 1)
	if err == nil {
		t.Fatal("queryOptimizationList() unexpectedly succeeded for an unbounded result")
	}
	if got, want := requests, appleads.MaxPlatformPaginationPages; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
	if got, want := len(items), appleads.MaxPlatformPaginationPages; got != want {
		t.Fatalf("accumulated items = %d, want the %d fetched pages returned alongside the error", got, want)
	}
	for _, want := range []string{"1000-page safety limit", "suggestions keywords find", "narrow"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to contain %q", err, want)
		}
	}
}

func TestQueryOptimizationRowsCapsUnboundedPages(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeJSON(t, w, `{"result":{"rows":[{"searchTerm":"habit tracker"}]},"pagination":{"offset":0,"pageSize":1}}`)
	}))
	defer server.Close()
	client, err := appleads.NewClient(appleads.Credentials{AccessToken: "token", AdAccountID: "account-1"}, appleads.WithPlatformBaseURL(server.URL+"/v1/"))
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := appleads.PlatformEndpointByCommandPath("insights", "search-term-popularity", "find")
	if !ok {
		t.Fatal("missing endpoint spec")
	}

	items, err := queryOptimizationRows[popularityResponse](context.Background(), client, spec, map[string]any{}, 1)
	if err == nil {
		t.Fatal("queryOptimizationRows() unexpectedly succeeded for an unbounded result")
	}
	if got, want := requests, appleads.MaxPlatformPaginationPages; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
	if got, want := len(items), appleads.MaxPlatformPaginationPages; got != want {
		t.Fatalf("accumulated rows = %d, want the %d fetched pages returned alongside the error", got, want)
	}
	for _, want := range []string{"1000-page safety limit", "insights search-term-popularity find", "narrow"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to contain %q", err, want)
		}
	}
}

func TestFetchOptimizationPhraseSuggestionsReadsPhraseField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode phrase-suggestion request: %v", err)
			http.Error(w, "invalid test request", http.StatusBadRequest)
			return
		}
		if hasOptimizationFilter(body, "countriesOrRegions") {
			t.Errorf("phrase suggestion request unexpectedly includes countriesOrRegions: %#v", body)
		}
		writeJSON(t, w, `{"result":[{"phrase":"best habit tracker","popularity":82}],"pagination":{"offset":0,"pageSize":1000,"totalCount":1}}`)
	}))
	defer server.Close()
	client, err := appleads.NewClient(
		appleads.Credentials{AccessToken: "token", AdAccountID: "account-1"},
		appleads.WithPlatformBaseURL(server.URL+"/v1/"),
	)
	if err != nil {
		t.Fatal(err)
	}

	items, err := fetchOptimizationSuggestions(context.Background(), client, SearchOptimizationRequest{
		AppID: "123456789", Country: "US",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Text != "best habit tracker" || items[0].Kind != "phrase" || items[0].Popularity == nil || *items[0].Popularity != 82 {
		t.Fatalf("phrase suggestions = %+v", items)
	}
}

func hasOptimizationFilter(body map[string]any, field string) bool {
	filters, _ := body["filters"].([]any)
	for _, raw := range filters {
		filter, _ := raw.(map[string]any)
		if filter["field"] == field {
			return true
		}
	}
	return false
}

func TestExecuteOptimizationQueryAppliesRequestTimeout(t *testing.T) {
	deadlinePresent := false
	client, err := appleads.NewClient(
		appleads.Credentials{AccessToken: "token", AdAccountID: "account-1"},
		appleads.WithHTTPClient(&http.Client{Transport: searchOptimizationRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			_, deadlinePresent = request.Context().Deadline()
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"result":[],"pagination":{"totalCount":0}}`)),
			}, nil
		})}),
	)
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := appleads.PlatformEndpointByCommandPath("suggestions", "keywords", "find")
	if !ok {
		t.Fatal("missing endpoint spec")
	}
	if _, err := executeOptimizationQuery(context.Background(), client, spec, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if !deadlinePresent {
		t.Fatal("optimization request context has no deadline")
	}
}

func TestFetchSearchOptimizationDataFailsWhenEveryIntelligenceSourceIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"message":"unavailable"}]}`, http.StatusBadRequest)
	}))
	defer server.Close()
	client, err := appleads.NewClient(
		appleads.Credentials{AccessToken: "token", AdAccountID: "account-1"},
		appleads.WithPlatformBaseURL(server.URL+"/v1/"),
	)
	if err != nil {
		t.Fatal(err)
	}

	data, err := fetchSearchOptimizationData(context.Background(), client, SearchOptimizationRequest{
		AppID: "123456789", Country: "US", Genre: "PRODUCTIVITY_UTILITIES",
		Start: "2026-07-19", End: "2026-08-17", PopularityStart: "2026-08-09", PopularityEnd: "2026-08-15",
	})
	if err == nil || !strings.Contains(err.Error(), "all official Apple Ads optimization sources are unavailable") {
		t.Fatalf("error = %v", err)
	}
	if source := findOptimizationSource(t, data.Sources, "search_term_performance"); source.Status != "unavailable" || !strings.Contains(source.Error, "campaign scope unavailable") {
		t.Fatalf("search term source = %+v", source)
	}
}

func assertFilter(t *testing.T, body map[string]any, field string, want any) {
	t.Helper()
	filters, _ := body["filters"].([]any)
	for _, raw := range filters {
		filter, _ := raw.(map[string]any)
		if filter["field"] != field {
			continue
		}
		value := filter["value"]
		switch typed := value.(type) {
		case []any:
			for _, item := range typed {
				if item == want {
					return
				}
			}
		default:
			if typed == want {
				return
			}
		}
	}
	t.Errorf("body filters do not contain %s=%v: %#v", field, want, body)
}

func assertOptimizationPaginationShape(t *testing.T, path string, body map[string]any) {
	t.Helper()
	if path == "/v1/suggestions/target-cpas/query" {
		if _, present := body["pagination"]; present {
			t.Errorf("%s body has unnecessary pagination: %#v", path, body)
		}
		return
	}
	pagination, ok := body["pagination"].(map[string]any)
	if !ok {
		t.Errorf("%s body has no pagination object: %#v", path, body)
		return
	}
	if _, ok := pagination["offset"]; !ok {
		t.Errorf("%s pagination has no offset: %#v", path, pagination)
	}
	if _, ok := pagination["pageSize"]; !ok {
		t.Errorf("%s pagination has no pageSize: %#v", path, pagination)
	}
	_, hasFetchTotalCount := pagination["fetchTotalCount"]
	expectsFetchTotalCount := path == "/v1/campaigns/query" || path == "/v1/keywords/query" || path == "/v1/negative-keywords/query"
	if hasFetchTotalCount != expectsFetchTotalCount {
		t.Errorf("%s pagination fetchTotalCount present = %t, want %t: %#v", path, hasFetchTotalCount, expectsFetchTotalCount, pagination)
	}
}

func assertNestedValue(t *testing.T, body map[string]any, object, field string, want any) {
	t.Helper()
	nested, ok := body[object].(map[string]any)
	if !ok || nested[field] != want {
		t.Errorf("%s.%s = %#v, want %#v in body %#v", object, field, nested[field], want, body)
	}
}

func assertSorting(t *testing.T, body map[string]any, field, orderKey, order string) {
	t.Helper()
	sorting, ok := body["sorting"].([]any)
	if !ok || len(sorting) != 1 {
		t.Errorf("sorting = %#v, want one entry", body["sorting"])
		return
	}
	entry, ok := sorting[0].(map[string]any)
	if !ok || entry["field"] != field || entry[orderKey] != order {
		t.Errorf("sorting = %#v, want %s %s=%s", sorting, field, orderKey, order)
	}
}

func assertArrayContains(t *testing.T, body map[string]any, field, want string) {
	t.Helper()
	items, ok := body[field].([]any)
	if !ok {
		t.Errorf("%s = %#v, want array containing %q", field, body[field], want)
		return
	}
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Errorf("%s = %#v, want array containing %q", field, items, want)
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(payload))
}

func findOptimizationSource(t *testing.T, sources []SearchOptimizationSourceStatus, name string) SearchOptimizationSourceStatus {
	t.Helper()
	for _, source := range sources {
		if source.Name == name {
			return source
		}
	}
	t.Fatalf("missing source %q in %+v", name, sources)
	return SearchOptimizationSourceStatus{}
}
