package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type adsRoundTripFunc func(*http.Request) (*http.Response, error)

func (f adsRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAdsCampaignsAliasPaginatesWithOrgContext(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))

	log := newRequestLog(2)
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.searchads.apple.com" {
			t.Fatalf("unexpected host %s", req.URL.Host)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer ACCESS" {
			t.Fatalf("Authorization = %q, want Bearer ACCESS", got)
		}
		if got := req.Header.Get("X-AP-Context"); got != "orgId=123456" {
			t.Fatalf("X-AP-Context = %q, want orgId=123456", got)
		}
		log.Add(req.URL.Path + "?" + req.URL.RawQuery)
		switch req.URL.Query().Get("offset") {
		case "0":
			return adsJSONResponse(200, `{"data":[{"id":1},{"id":2}],"pagination":{"itemsPerPage":2,"startIndex":0,"totalResults":3}}`), nil
		case "2":
			return adsJSONResponse(200, `{"data":[{"id":3}],"pagination":{"itemsPerPage":2,"startIndex":2,"totalResults":3}}`), nil
		default:
			t.Fatalf("unexpected offset %q", req.URL.Query().Get("offset"))
			return nil, nil
		}
	}))

	root := RootCommand("dev")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"ads", "v5", "campaigns", "--limit", "2", "--paginate", "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if got, want := stderr, adsV5ReplacementWarning("v5 campaigns", "campaigns find"); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	var parsed struct {
		Data []map[string]int `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	if len(parsed.Data) != 3 || parsed.Data[2]["id"] != 3 {
		t.Fatalf("data = %+v, want three aggregated campaign rows", parsed.Data)
	}
	requests := strings.Join(log.Snapshot(), "\n")
	if !strings.Contains(requests, "/api/v5/campaigns?limit=2&offset=0") || !strings.Contains(requests, "/api/v5/campaigns?limit=2&offset=2") {
		t.Fatalf("requests = %q, want both paginated offsets", requests)
	}
}

func TestAdsReportsPresetBuildsCampaignRequest(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))

	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v5/reports/campaigns" {
			t.Fatalf("request = %s %s, want POST /api/v5/reports/campaigns", req.Method, req.URL.String())
		}
		if got := req.Header.Get("X-AP-Context"); got != "orgId=123456" {
			t.Fatalf("X-AP-Context = %q, want orgId=123456", got)
		}
		var body struct {
			StartTime       string `json:"startTime"`
			EndTime         string `json:"endTime"`
			Granularity     string `json:"granularity"`
			ReturnRowTotals bool   `json:"returnRowTotals"`
			TimeZone        string `json:"timeZone"`
			Selector        struct {
				Fields  []string `json:"fields"`
				OrderBy []struct {
					Field     string `json:"field"`
					SortOrder string `json:"sortOrder"`
				} `json:"orderBy"`
				Pagination struct {
					Offset int `json:"offset"`
					Limit  int `json:"limit"`
				} `json:"pagination"`
			} `json:"selector"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		startDate, err := time.Parse("2006-01-02", body.StartTime)
		if err != nil {
			t.Fatalf("startTime = %q, want YYYY-MM-DD", body.StartTime)
		}
		endDate, err := time.Parse("2006-01-02", body.EndTime)
		if err != nil {
			t.Fatalf("endTime = %q, want YYYY-MM-DD", body.EndTime)
		}
		if endDate.Sub(startDate) != 6*24*time.Hour {
			t.Fatalf("date range = %s..%s, want 7-day hourly window", body.StartTime, body.EndTime)
		}
		if body.Granularity != "HOURLY" || body.TimeZone != "UTC" || !body.ReturnRowTotals {
			t.Fatalf("report options = %+v, want hourly UTC totals", body)
		}
		if strings.Join(body.Selector.Fields, ",") != "campaignName,impressions,taps,localSpend" {
			t.Fatalf("fields = %v", body.Selector.Fields)
		}
		if len(body.Selector.OrderBy) != 1 || body.Selector.OrderBy[0].Field != "impressions" || body.Selector.OrderBy[0].SortOrder != "DESCENDING" {
			t.Fatalf("orderBy = %+v, want impressions descending", body.Selector.OrderBy)
		}
		if body.Selector.Pagination.Offset != 5 || body.Selector.Pagination.Limit != 25 {
			t.Fatalf("pagination = %+v, want offset 5 limit 25", body.Selector.Pagination)
		}
		return adsJSONResponse(200, `{"data":{"reportingDataResponse":{"row":[{"metadata":{"campaignId":12345},"total":{"impressions":42}}]}}}`), nil
	}))

	root := RootCommand("dev")
	args := []string{
		"ads", "v5", "reports", "preset",
		"--level", "campaigns",
		"--last-days", "7",
		"--fields", "campaignName,impressions,taps,spend",
		"--granularity", "hourly",
		"--sort", "-impressions",
		"--limit", "25",
		"--offset", "5",
		"--return-row-totals",
		"--output", "json",
	}
	if err := root.Parse(args); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if got, want := stderr, adsV5ReplacementWarning("v5 reports preset", "reports apps campaigns"); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
}

func TestAdsReportsPresetBuildsScopedKeywordRequest(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	from, to := adsReportRecentRange(7)

	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v5/reports/campaigns/12345/keywords" {
			t.Fatalf("request = %s %s, want keyword report path", req.Method, req.URL.String())
		}
		if got := req.Header.Get("X-AP-Context"); got != "orgId=987654" {
			t.Fatalf("X-AP-Context = %q, want explicit org", got)
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["startTime"] != from || body["endTime"] != to {
			t.Fatalf("date range = %#v..%#v, want %s..%s", body["startTime"], body["endTime"], from, to)
		}
		return adsJSONResponse(200, `{"data":{"reportingDataResponse":{"row":[]}}}`), nil
	}))

	root := RootCommand("dev")
	args := []string{
		"ads", "v5", "reports", "preset",
		"--level", "keywords",
		"--campaign", "12345",
		"--from", from,
		"--to", to,
		"--org", "987654",
		"--output", "json",
	}
	if err := root.Parse(args); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if got, want := stderr, adsV5ReplacementWarning("v5 reports preset", "reports apps keywords"); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
}

func TestAdsReportsPresetBuildsAdLevelRequestWithSort(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	from, to := adsReportRecentRange(7)

	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v5/reports/campaigns/12345/ads" {
			t.Fatalf("request = %s %s, want ad report path", req.Method, req.URL.String())
		}
		var body struct {
			Selector struct {
				OrderBy []struct {
					Field     string `json:"field"`
					SortOrder string `json:"sortOrder"`
				} `json:"orderBy"`
			} `json:"selector"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body.Selector.OrderBy) != 1 || body.Selector.OrderBy[0].Field != "impressions" || body.Selector.OrderBy[0].SortOrder != "DESCENDING" {
			t.Fatalf("orderBy = %+v, want impressions descending", body.Selector.OrderBy)
		}
		return adsJSONResponse(200, `{"data":{"reportingDataResponse":{"row":[]}}}`), nil
	}))

	root := RootCommand("dev")
	args := []string{
		"ads", "v5", "reports", "preset",
		"--level", "ads",
		"--campaign", "12345",
		"--from", from,
		"--to", to,
		"--sort", "-impressions",
		"--output", "json",
	}
	if err := root.Parse(args); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if got, want := stderr, adsV5ReplacementWarning("v5 reports preset", "reports apps ads"); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
}

func TestAdsReportsPresetValidatesUsageBeforeNetwork(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	recentFrom, recentTo := adsReportRecentRange(7)
	hourlyLongFrom, hourlyLongTo := adsReportRangeEnding(8, 0)
	hourlyOldFrom, hourlyOldTo := adsReportRangeEnding(31, 25)
	dailyLongFrom, dailyLongTo := adsReportRangeEnding(91, 0)
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing date range",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "campaigns", "--output", "json"},
			wantErr: "either --last-days or both --from and --to are required",
		},
		{
			name:    "invalid level",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "unsupported", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--level must be one of:",
		},
		{
			name:    "campaign required",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "keywords", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--campaign is required for --level keywords",
		},
		{
			name:    "campaign nonnegative",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "keywords", "--campaign", "-1", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--campaign must be >= 0",
		},
		{
			name:    "campaign unsupported for campaign level",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "campaigns", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--campaign is not supported for --level campaigns",
		},
		{
			name:    "ad group unsupported for keyword level",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "keywords", "--campaign", "12345", "--ad-group", "67890", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--ad-group is not supported for --level keywords",
		},
		{
			name:    "invalid sort direction",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "campaigns", "--from", recentFrom, "--to", recentTo, "--sort", "impressions:sideways", "--output", "json"},
			wantErr: "--sort direction must be asc or desc",
		},
		{
			name:    "invalid granularity",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "campaigns", "--from", recentFrom, "--to", recentTo, "--granularity", "YEARLY", "--output", "json"},
			wantErr: "--granularity must be one of: HOURLY, DAILY, WEEKLY, MONTHLY",
		},
		{
			name:    "hourly unsupported for search terms",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "search-terms", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--granularity", "HOURLY", "--output", "json"},
			wantErr: "--granularity HOURLY is only supported",
		},
		{
			name:    "hourly unsupported for ads",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "ads", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--granularity", "HOURLY", "--sort", "-impressions", "--output", "json"},
			wantErr: "--granularity HOURLY is only supported",
		},
		{
			name:    "hourly range too long",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "campaigns", "--from", hourlyLongFrom, "--to", hourlyLongTo, "--granularity", "HOURLY", "--output", "json"},
			wantErr: "--granularity HOURLY supports a maximum 7-day date range",
		},
		{
			name:    "hourly start too old",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "campaigns", "--from", hourlyOldFrom, "--to", hourlyOldTo, "--granularity", "HOURLY", "--output", "json"},
			wantErr: "--granularity HOURLY start date must be within the last 30 days",
		},
		{
			name:    "daily range too long",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "campaigns", "--from", dailyLongFrom, "--to", dailyLongTo, "--granularity", "DAILY", "--output", "json"},
			wantErr: "--granularity DAILY supports a maximum 90-day date range",
		},
		{
			name:    "row totals unsupported for search terms",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "search-terms", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--return-row-totals", "--output", "json"},
			wantErr: "--return-row-totals cannot be used with search-term report levels",
		},
		{
			name:    "invalid time zone",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "campaigns", "--last-days", "1", "--time-zone", "America/Los_Angeles", "--output", "json"},
			wantErr: "--time-zone must be UTC or ORTZ",
		},
		{
			name:    "search terms require explicit ORTZ",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "search-terms", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--time-zone", "UTC", "--output", "json"},
			wantErr: "--time-zone must be ORTZ for search-term report levels",
		},
		{
			name:    "last days unsupported for ORTZ",
			args:    []string{"ads", "v5", "reports", "preset", "--level", "campaigns", "--last-days", "1", "--time-zone", "ORTZ", "--output", "json"},
			wantErr: "--last-days is not supported for ORTZ reports; use --from and --to",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := RootCommand("dev")
			if err := root.Parse(tt.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			var runErr error
			_, stderr := captureOutput(t, func() {
				runErr = root.Run(context.Background())
			})
			if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, tt.wantErr) {
				t.Fatalf("run error = %v stderr = %q, want %q", runErr, stderr, tt.wantErr)
			}
		})
	}
}

func TestAdsImpressionShareReportsLimitValidation(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "v5", "impression-share-reports", "--limit", "51", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "--limit must be between 1 and 50") {
		t.Fatalf("run error = %v stderr = %q, want custom reports limit validation", runErr, stderr)
	}
}

func adsReportRecentRange(days int) (string, string) {
	return adsReportRangeEnding(days-1, 0)
}

func adsReportRangeEnding(startDaysAgo, endDaysAgo int) (string, string) {
	now := time.Now().UTC()
	return now.AddDate(0, 0, -startDaysAgo).Format("2006-01-02"), now.AddDate(0, 0, -endDaysAgo).Format("2006-01-02")
}

func TestAdsLimitZeroValidation(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "v5", "campaigns", "--limit", "0", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "--limit must be between 1 and 1000") {
		t.Fatalf("run error = %v stderr = %q, want zero limit validation", runErr, stderr)
	}
}

func TestAdsDeleteRequiresConfirmBeforeNetwork(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "v5", "campaigns", "delete", "--campaign", "123"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "--confirm is required") {
		t.Fatalf("run error = %v stderr = %q, want confirm validation", runErr, stderr)
	}
}

func TestAdsV5RiskMutationsRequireConfirmBeforeFileAuthOrNetwork(t *testing.T) {
	missingPayload := filepath.Join(t.TempDir(), "missing.json")
	tests := []struct {
		name string
		args []string
	}{
		{name: "budget create", args: []string{"ads", "v5", "budget-orders", "create", "--file", missingPayload}},
		{name: "budget update", args: []string{"ads", "v5", "budget-orders", "update", "--budget-order", "1", "--file", missingPayload}},
		{name: "campaign create", args: []string{"ads", "v5", "campaigns", "create", "--file", missingPayload}},
		{name: "campaign update", args: []string{"ads", "v5", "campaigns", "update", "--campaign", "1", "--file", missingPayload}},
		{name: "ad group create", args: []string{"ads", "v5", "ad-groups", "create", "--campaign", "1", "--file", missingPayload}},
		{name: "ad group update", args: []string{"ads", "v5", "ad-groups", "update", "--campaign", "1", "--ad-group", "2", "--file", missingPayload}},
		{name: "ad create", args: []string{"ads", "v5", "ads", "create", "--campaign", "1", "--ad-group", "2", "--file", missingPayload}},
		{name: "ad update", args: []string{"ads", "v5", "ads", "update", "--campaign", "1", "--ad-group", "2", "--ad", "3", "--file", missingPayload}},
		{name: "targeting keyword create", args: []string{"ads", "v5", "targeting-keywords", "create-bulk", "--campaign", "1", "--ad-group", "2", "--file", missingPayload}},
		{name: "targeting keyword update", args: []string{"ads", "v5", "targeting-keywords", "update-bulk", "--campaign", "1", "--ad-group", "2", "--file", missingPayload}},
		{name: "campaign negative keyword create", args: []string{"ads", "v5", "campaign-negative-keywords", "create-bulk", "--campaign", "1", "--file", missingPayload}},
		{name: "campaign negative keyword update", args: []string{"ads", "v5", "campaign-negative-keywords", "update-bulk", "--campaign", "1", "--file", missingPayload}},
		{name: "ad group negative keyword create", args: []string{"ads", "v5", "ad-group-negative-keywords", "create-bulk", "--campaign", "1", "--ad-group", "2", "--file", missingPayload}},
		{name: "ad group negative keyword update", args: []string{"ads", "v5", "ad-group-negative-keywords", "update-bulk", "--campaign", "1", "--ad-group", "2", "--file", missingPayload}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isolateAdsGuideEnv(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing-config.json"))
			installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("unexpected network request before confirmation: %s %s", req.Method, req.URL.String())
				return nil, nil
			}))

			root := RootCommand("dev")
			if err := root.Parse(test.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			var runErr error
			stdout, stderr := captureOutput(t, func() {
				runErr = root.Run(context.Background())
			})
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "--confirm is required") {
				t.Fatalf("run error = %v stderr = %q, want pre-auth confirmation error", runErr, stderr)
			}
			if strings.Contains(stderr, "configuration not found") || strings.Contains(stderr, "missing.json") {
				t.Fatalf("stderr = %q, confirmation must precede file and auth resolution", stderr)
			}
		})
	}
}

func TestAdsV5RawRequestSafetyGuardsPrecedeAuthAndNetwork(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "known spend mutation",
			args:    []string{"ads", "v5", "api", "request", "--method", "POST", "--path", "v5/campaigns"},
			wantErr: "--confirm is required",
		},
		{
			name:    "known bulk delete",
			args:    []string{"ads", "v5", "api", "request", "--method", "POST", "--path", "v5/campaigns/1/negativekeywords/delete/bulk"},
			wantErr: "--confirm is required",
		},
		{
			name:    "unknown POST fails closed",
			args:    []string{"ads", "v5", "api", "request", "--method", "POST", "--path", "v5/future-resource/query"},
			wantErr: "--confirm is required",
		},
		{
			name:    "unknown PUT fails closed",
			args:    []string{"ads", "v5", "api", "request", "--method", "PUT", "--path", "v5/future-resource/1"},
			wantErr: "--confirm is required",
		},
		{
			name:    "invalid output",
			args:    []string{"ads", "v5", "api", "request", "--method", "GET", "--path", "v5/campaigns", "--output", "yaml"},
			wantErr: `(got "yaml")`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isolateAdsGuideEnv(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing-config.json"))
			installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("unexpected network request before safety validation: %s %s", req.Method, req.URL.String())
				return nil, nil
			}))

			root := RootCommand("dev")
			if err := root.Parse(test.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			var runErr error
			stdout, stderr := captureOutput(t, func() {
				runErr = root.Run(context.Background())
			})
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !isUsageClassError(runErr) || !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("run error = %v stderr = %q, want %q before auth", runErr, stderr, test.wantErr)
			}
			if strings.Contains(stderr, "configuration not found") {
				t.Fatalf("stderr = %q, safety validation must precede auth resolution", stderr)
			}
		})
	}
}

func TestAdsV5RawKnownReadLikePostDoesNotRequireConfirm(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing-config.json"))

	requestCount := 0
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if req.Method != http.MethodPost || req.URL.Path != "/api/v5/campaigns/find" {
			t.Fatalf("request = %s %s, want POST /api/v5/campaigns/find", req.Method, req.URL.String())
		}
		return adsJSONResponse(http.StatusOK, `{"data":[]}`), nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{
		"ads", "v5", "api", "request",
		"--method", "POST",
		"--path", "v5/campaigns/find",
		"--output", "json",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("run error = %v stderr = %q", runErr, stderr)
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
	if !strings.Contains(stdout, `"data":[]`) {
		t.Fatalf("stdout = %q, want raw API response", stdout)
	}
	if got, want := stderr, adsV5ReplacementWarning("v5 api request", "api request"); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestAdsCampaignPauseAndResumeUseCuratedStatusPayloads(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))

	log := newRequestLog(2)
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut || req.URL.Path != "/api/v5/campaigns/123" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		if got := req.Header.Get("X-AP-Context"); got != "orgId=123456" {
			t.Fatalf("X-AP-Context = %q, want orgId=123456", got)
		}
		var body struct {
			Campaign struct {
				Status string `json:"status"`
			} `json:"campaign"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		status := body.Campaign.Status
		log.Add(status)
		return adsJSONResponse(200, `{"data":{"id":123,"status":"`+status+`"}}`), nil
	}))

	for _, args := range [][]string{
		{"ads", "v5", "campaigns", "pause", "--campaign", "123", "--confirm", "--output", "json"},
		{"ads", "v5", "campaigns", "resume", "--campaign", "123", "--confirm", "--output", "json"},
	} {
		root := RootCommand("dev")
		if err := root.Parse(args); err != nil {
			t.Fatalf("parse %s: %v", strings.Join(args, " "), err)
		}
		stdout, stderr := captureOutput(t, func() {
			if err := root.Run(context.Background()); err != nil {
				t.Fatalf("run %s: %v", strings.Join(args, " "), err)
			}
		})
		wantWarning := adsV5ReplacementWarning("v5 campaigns "+args[3], "campaigns "+args[3])
		if got, want := stderr, wantWarning; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		var parsed struct {
			Data struct {
				ID     int    `json:"id"`
				Status string `json:"status"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
			t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
		}
		if parsed.Data.ID != 123 || parsed.Data.Status == "" {
			t.Fatalf("parsed data = %+v, want campaign status response", parsed.Data)
		}
	}

	requests := strings.Join(log.Snapshot(), "\n")
	if requests != "PAUSED\nENABLED" {
		t.Fatalf("payload statuses = %q, want PAUSED then ENABLED", requests)
	}
}

func TestAdsCampaignPauseHonorsParentFlagsBeforeWorkflowSubcommand(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))

	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut || req.URL.Path != "/api/v5/campaigns/123" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		if got := req.Header.Get("X-AP-Context"); got != "orgId=123456" {
			t.Fatalf("X-AP-Context = %q, want parent --org value", got)
		}
		return adsJSONResponse(200, `{"data":{"id":123,"status":"PAUSED"}}`), nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "v5", "campaigns", "--org", "123456", "pause", "--campaign", "123", "--confirm"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if got, want := stderr, adsV5ReplacementWarning("v5 campaigns pause", "campaigns pause"); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if !strings.Contains(stdout, `"status":"PAUSED"`) {
		t.Fatalf("stdout = %q, want paused response", stdout)
	}
}

func TestAdsCampaignPauseValidatesBeforeNetwork(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	for _, tc := range []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing confirm",
			args:    []string{"ads", "v5", "campaigns", "pause", "--campaign", "123"},
			wantErr: "--confirm is required",
		},
		{
			name:    "invalid campaign",
			args:    []string{"ads", "v5", "campaigns", "pause", "--campaign", "abc", "--confirm"},
			wantErr: "--campaign must be an integer",
		},
		{
			name:    "missing campaign",
			args:    []string{"ads", "v5", "campaigns", "pause", "--confirm"},
			wantErr: "--campaign is required",
		},
		{
			name:    "parent output conflicts with child pretty",
			args:    []string{"ads", "v5", "campaigns", "--output", "table", "pause", "--campaign", "123", "--confirm", "--pretty"},
			wantErr: `(got "table")`,
		},
		{
			name:    "parent pretty conflicts with child output",
			args:    []string{"ads", "v5", "campaigns", "--pretty", "resume", "--campaign", "123", "--confirm", "--output", "table"},
			wantErr: `(got "table")`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RootCommand("dev")
			if err := root.Parse(tc.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			var runErr error
			_, stderr := captureOutput(t, func() {
				runErr = root.Run(context.Background())
			})
			if !isUsageClassError(runErr) || !strings.Contains(stderr, tc.wantErr) {
				t.Fatalf("run error = %v stderr = %q, want %q", runErr, stderr, tc.wantErr)
			}
		})
	}
}

func TestAdsCampaignResumeReportsCommandNameOnAuthFailure(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "v5", "campaigns", "resume", "--campaign", "123", "--confirm", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if runErr == nil || !strings.Contains(runErr.Error(), "ads v5 campaigns resume:") {
		t.Fatalf("run error = %v, want resume command name", runErr)
	}
	if got, want := stderr, adsV5ReplacementWarning("v5 campaigns resume", "campaigns resume"); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestAdsEndpointRejectsUnexpectedArgsBeforeNetwork(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "v5", "campaigns", "--output", "json", "unexpected"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "unexpected argument(s): unexpected") {
		t.Fatalf("run error = %v stderr = %q, want unexpected argument usage error", runErr, stderr)
	}
}

func TestAdsAPIRequestRejectsNonAppleURLsBeforeNetwork(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "v5", "api", "request", "--path", "https://example.com/api/v5/campaigns"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "Apple Ads v5 URL") {
		t.Fatalf("run error = %v stderr = %q, want Apple host guardrail", runErr, stderr)
	}
}

func TestAdsAPIRequestRejectsUnexpectedArgsBeforeNetwork(t *testing.T) {
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "123456")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "v5", "api", "request", "--path", "v5/campaigns", "--output", "json", "unexpected"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "unexpected argument(s): unexpected") {
		t.Fatalf("run error = %v stderr = %q, want unexpected argument usage error", runErr, stderr)
	}
}

func TestAdsPlatformAPIRequestUsesV1HostAndAdAccountContext(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_AD_ACCOUNT_ID", "123")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Host != "api.ads.apple.com" || req.URL.Path != "/v1/ad-accounts/123" {
			t.Fatalf("request = %s %s", req.Method, req.URL.String())
		}
		if got := req.Header.Get("Authorization"); got != "Bearer ACCESS" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := req.Header.Get("X-AP-Context"); got != "adAccountId=123;" {
			t.Fatalf("X-AP-Context = %q", got)
		}
		return adsJSONResponse(200, `{"result":{"id":"123"}}`), nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "api", "request", "--path", "v1/ad-accounts/123", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
	var output struct {
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	if output.Result.ID != "123" {
		t.Fatalf("result = %+v, want preserved v1 result envelope with id 123", output.Result)
	}
}

func TestAdsPlatformAPIRequestRejectsAdAccountPathMismatchBeforeNetwork(t *testing.T) {
	for _, path := range []string{"v1/ad-accounts/PATH_ACCOUNT", "https://api.ads.apple.com/v1/ad-accounts/PATH_ACCOUNT"} {
		t.Run(path, func(t *testing.T) {
			isolateAdsGuideEnv(t)
			t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
			t.Setenv("ASC_ADS_AD_ACCOUNT_ID", "CONTEXT_ACCOUNT")
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
			installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("unexpected token/network request: %s %s", req.Method, req.URL.String())
				return nil, nil
			}))

			stdout, stderr, err := runAdsEvalCommand(t, "ads", "api", "request", "--method", "GET", "--path", path, "--output", "json")
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("error = %v, want usage error", err)
			}
			if stdout != "" || !strings.Contains(stderr, "must match the v1/ad-accounts path ID") {
				t.Fatalf("stdout=%q stderr=%q error=%v, want path/context mismatch before network", stdout, stderr, err)
			}
		})
	}
}

func TestAdsPlatformAPIRequestRejectsInvalidOutputBeforeNetwork(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	stdout, stderr, err := runAdsEvalCommand(t, "ads", "api", "request", "--path", "v1/me", "--output", "invalid")
	if !isUsageClassError(err) || stdout != "" || !strings.Contains(stderr, `(got "invalid")`) {
		t.Fatalf("stdout=%q stderr=%q error=%v, want preflight output error", stdout, stderr, err)
	}
}

func TestAdsPlatformAPIRequestRejectsMultipartUploadBeforeAuthOrNetwork(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_AD_ACCOUNT_ID", "AD_ACCOUNT")
	t.Setenv("ASC_ADS_ORG_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected token/network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	stdout, stderr, err := runAdsEvalCommand(t, "ads", "api", "request", "--method", "POST", "--path", "v1/assets/upload", "--file", filepath.Join(t.TempDir(), "payload.json"))
	if !errors.Is(err, flag.ErrHelp) || stdout != "" {
		t.Fatalf("stdout=%q stderr=%q error=%v, want preflight usage error", stdout, stderr, err)
	}
	for _, want := range []string{"multipart/form-data", "asc ads assets upload"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr=%q, want %q", stderr, want)
		}
	}
}

func TestAdsEndpointRejectsInvalidOutputBeforeReadingBody(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	stdout, stderr, err := runAdsEvalCommand(t, "ads", "ad-accounts", "create", "--file", filepath.Join(t.TempDir(), "does-not-exist.json"), "--output", "invalid")
	if !isUsageClassError(err) || stdout != "" || !strings.Contains(stderr, `(got "invalid")`) {
		t.Fatalf("stdout=%q stderr=%q error=%v, want output preflight before body read", stdout, stderr, err)
	}
}

func TestAdsPlatformAPIRequestOmitsContextForMe(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_AD_ACCOUNT_ID", "AD_ACCOUNT")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.ads.apple.com" || req.URL.Path != "/v1/me" {
			t.Fatalf("request URL = %s", req.URL.String())
		}
		if got := req.Header.Get("X-AP-Context"); got != "" {
			t.Fatalf("X-AP-Context = %q, want empty", got)
		}
		return adsJSONResponse(200, `{"result":{"userId":"1"}}`), nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "api", "request", "--path", "v1/me", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestAdsPlatformAPIRequestRequiresAdAccountBeforeNetwork(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_AD_ACCOUNT_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "api", "request", "--path", "v1/campaigns/123"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "--ad-account is required") {
		t.Fatalf("run error = %v stderr = %q", runErr, stderr)
	}
}

func TestAdsPlatformAPIRequestRejectsAdAccountForContextFreeEndpoint(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "api", "request", "--path", "v1/me", "--ad-account", "AD_ACCOUNT"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "--ad-account is not supported") {
		t.Fatalf("run error = %v stderr = %q", runErr, stderr)
	}
}

func TestAdsPlatformAPIRequestRequiresConfirmForKnownImpactMutations(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "daily budget apply",
			args: []string{"ads", "api", "request", "--method", "POST", "--path", "v1/recommendations/daily-budgets/apply", "--ad-account", "AD_ACCOUNT"},
		},
		{
			name: "daily budget dismiss",
			args: []string{"ads", "api", "request", "--method", "POST", "--path", "v1/recommendations/daily-budgets/dismiss", "--ad-account", "AD_ACCOUNT"},
		},
		{
			name: "target CPA apply",
			args: []string{"ads", "api", "request", "--method", "POST", "--path", "v1/recommendations/target-cpas/apply", "--ad-account", "AD_ACCOUNT"},
		},
		{
			name: "target CPA dismiss",
			args: []string{"ads", "api", "request", "--method", "POST", "--path", "v1/recommendations/target-cpas/dismiss", "--ad-account", "AD_ACCOUNT"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateAdsGuideEnv(t)
			t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
			t.Setenv("ASC_ADS_AD_ACCOUNT_ID", "AD_ACCOUNT")
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
			installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
				return nil, nil
			}))

			root := RootCommand("dev")
			if err := root.Parse(tt.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			var runErr error
			_, stderr := captureOutput(t, func() {
				runErr = root.Run(context.Background())
			})
			want := "--confirm is required to acknowledge potential Apple Ads spend, billing, delivery, targeting, or access impact"
			if !errors.Is(runErr, flag.ErrHelp) || runErr.Error() != want || !strings.Contains(stderr, want) {
				t.Fatalf("run error = %v stderr = %q", runErr, stderr)
			}
		})
	}
}

func TestAdsPlatformAPIRequestRequiresConfirmForDelegationReplacement(t *testing.T) {
	tempDir := t.TempDir()
	payloadPath := filepath.Join(tempDir, "delegations.json")
	if err := os.WriteFile(payloadPath, []byte(`{"delegations":[{"resourceId":"RESOURCE","resourceType":"CONTENT_PROVIDER"}]}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_AD_ACCOUNT_ID", "AD_ACCOUNT")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(tempDir, "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{
		"ads", "api", "request",
		"--method", "PUT",
		"--path", "v1/ad-accounts/AD_ACCOUNT",
		"--file", payloadPath,
		"--ad-account", "AD_ACCOUNT",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || runErr.Error() != "--confirm is required" || !strings.Contains(stderr, "--confirm is required") {
		t.Fatalf("run error = %v stderr = %q", runErr, stderr)
	}
}

func TestAdsPlatformAPIRequestDefersBodyDependentConfirmationUntilAfterPayload(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantMethod string
		wantPath   string
	}{
		{
			name:       "paused campaign create",
			method:     http.MethodPost,
			path:       "v1/campaigns",
			body:       `{"status":"PAUSED"}`,
			wantMethod: http.MethodPost,
			wantPath:   "/v1/campaigns",
		},
		{
			name:       "name-only ad-account update",
			method:     http.MethodPut,
			path:       "v1/ad-accounts/AD_ACCOUNT",
			body:       `{"name":"Renamed"}`,
			wantMethod: http.MethodPut,
			wantPath:   "/v1/ad-accounts/AD_ACCOUNT",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payloadPath := writeAdsEvalPayload(t, "payload.json", test.body)
			isolateAdsGuideEnv(t)
			t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
			t.Setenv("ASC_ADS_AD_ACCOUNT_ID", "AD_ACCOUNT")
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
			installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != test.wantMethod || req.URL.Path != test.wantPath {
					t.Fatalf("request = %s %s, want %s %s", req.Method, req.URL.Path, test.wantMethod, test.wantPath)
				}
				return adsJSONResponse(200, `{"data":{"id":"1"}}`), nil
			}))

			stdout, stderr, err := runAdsEvalCommand(
				t,
				"ads", "api", "request",
				"--method", test.method,
				"--path", test.path,
				"--file", payloadPath,
				"--ad-account", "AD_ACCOUNT",
				"--output", "json",
			)
			if err != nil {
				t.Fatalf("run error = %v, want body-dependent confirmation after payload read", err)
			}
			if stderr != "" || !strings.Contains(stdout, `"data"`) {
				t.Fatalf("stdout=%q stderr=%q, want successful raw response", stdout, stderr)
			}
		})
	}
}

func TestAdsPlatformAPIRequestAllowsKnownReadOnlyPostWithoutConfirmation(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_AD_ACCOUNT_ID", "AD_ACCOUNT")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/campaigns/query" {
			t.Fatalf("request = %s %s, want POST /v1/campaigns/query", req.Method, req.URL.Path)
		}
		return adsJSONResponse(200, `{"data":[]}`), nil
	}))

	stdout, stderr, err := runAdsEvalCommand(
		t,
		"ads", "api", "request",
		"--method", http.MethodPost,
		"--path", "v1/campaigns/query",
		"--ad-account", "AD_ACCOUNT",
		"--output", "json",
	)
	if err != nil {
		t.Fatalf("run error = %v, want known read-only POST to remain confirmation-free", err)
	}
	if stderr != "" || !strings.Contains(stdout, `"data"`) {
		t.Fatalf("stdout=%q stderr=%q, want successful raw response", stdout, stderr)
	}
}

func TestAdsPlatformAPIRequestKeepsUnconditionalConfirmationBeforePayloadRead(t *testing.T) {
	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	missingPayload := filepath.Join(t.TempDir(), "missing.json")
	stdout, stderr, err := runAdsEvalCommand(
		t,
		"ads", "api", "request",
		"--method", http.MethodPost,
		"--path", "v1/ad-accounts",
		"--file", missingPayload,
		"--output", "json",
	)
	if !errors.Is(err, flag.ErrHelp) || !strings.Contains(stderr, "--confirm is required to acknowledge potential Apple Ads spend, billing, delivery, targeting, or access impact") {
		t.Fatalf("stdout=%q stderr=%q error=%v, want unconditional confirmation before payload read", stdout, stderr, err)
	}
}

func TestAdsPlatformAdAccountCreateRequiresRiskConfirmationBeforeNetwork(t *testing.T) {
	payloadPath := filepath.Join(t.TempDir(), "ad-account.json")
	if err := os.WriteFile(payloadPath, []byte(`{"name":"Disposable","productFeatures":["APPSTORE_APP_MANUAL"]}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	isolateAdsGuideEnv(t)
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	installDefaultTransport(t, adsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected token/network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	}))

	root := RootCommand("dev")
	if err := root.Parse([]string{"ads", "ad-accounts", "create", "--file", payloadPath}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "--confirm is required to acknowledge potential Apple Ads spend, billing, delivery, targeting, or access impact") {
		t.Fatalf("run error = %v stderr = %q, want risk confirmation", runErr, stderr)
	}
}

func adsJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
