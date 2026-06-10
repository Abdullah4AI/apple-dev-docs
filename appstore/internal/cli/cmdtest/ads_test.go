package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
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
		if err := root.Parse([]string{"ads", "campaigns", "--limit", "2", "--paginate", "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
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
		"ads", "reports", "preset",
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
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
		"ads", "reports", "preset",
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
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
		"ads", "reports", "preset",
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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
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
			args:    []string{"ads", "reports", "preset", "--level", "campaigns", "--output", "json"},
			wantErr: "either --last-days or both --from and --to are required",
		},
		{
			name:    "invalid level",
			args:    []string{"ads", "reports", "preset", "--level", "unsupported", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--level must be one of:",
		},
		{
			name:    "campaign required",
			args:    []string{"ads", "reports", "preset", "--level", "keywords", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--campaign is required for --level keywords",
		},
		{
			name:    "campaign nonnegative",
			args:    []string{"ads", "reports", "preset", "--level", "keywords", "--campaign", "-1", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--campaign must be >= 0",
		},
		{
			name:    "campaign unsupported for campaign level",
			args:    []string{"ads", "reports", "preset", "--level", "campaigns", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--campaign is not supported for --level campaigns",
		},
		{
			name:    "ad group unsupported for keyword level",
			args:    []string{"ads", "reports", "preset", "--level", "keywords", "--campaign", "12345", "--ad-group", "67890", "--from", recentFrom, "--to", recentTo, "--output", "json"},
			wantErr: "--ad-group is not supported for --level keywords",
		},
		{
			name:    "invalid sort direction",
			args:    []string{"ads", "reports", "preset", "--level", "campaigns", "--from", recentFrom, "--to", recentTo, "--sort", "impressions:sideways", "--output", "json"},
			wantErr: "--sort direction must be asc or desc",
		},
		{
			name:    "invalid granularity",
			args:    []string{"ads", "reports", "preset", "--level", "campaigns", "--from", recentFrom, "--to", recentTo, "--granularity", "YEARLY", "--output", "json"},
			wantErr: "--granularity must be one of: HOURLY, DAILY, WEEKLY, MONTHLY",
		},
		{
			name:    "hourly unsupported for search terms",
			args:    []string{"ads", "reports", "preset", "--level", "search-terms", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--granularity", "HOURLY", "--output", "json"},
			wantErr: "--granularity HOURLY is only supported",
		},
		{
			name:    "hourly unsupported for ads",
			args:    []string{"ads", "reports", "preset", "--level", "ads", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--granularity", "HOURLY", "--sort", "-impressions", "--output", "json"},
			wantErr: "--granularity HOURLY is only supported",
		},
		{
			name:    "hourly range too long",
			args:    []string{"ads", "reports", "preset", "--level", "campaigns", "--from", hourlyLongFrom, "--to", hourlyLongTo, "--granularity", "HOURLY", "--output", "json"},
			wantErr: "--granularity HOURLY supports a maximum 7-day date range",
		},
		{
			name:    "hourly start too old",
			args:    []string{"ads", "reports", "preset", "--level", "campaigns", "--from", hourlyOldFrom, "--to", hourlyOldTo, "--granularity", "HOURLY", "--output", "json"},
			wantErr: "--granularity HOURLY start date must be within the last 30 days",
		},
		{
			name:    "daily range too long",
			args:    []string{"ads", "reports", "preset", "--level", "campaigns", "--from", dailyLongFrom, "--to", dailyLongTo, "--granularity", "DAILY", "--output", "json"},
			wantErr: "--granularity DAILY supports a maximum 90-day date range",
		},
		{
			name:    "row totals unsupported for search terms",
			args:    []string{"ads", "reports", "preset", "--level", "search-terms", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--return-row-totals", "--output", "json"},
			wantErr: "--return-row-totals cannot be used with search-term report levels",
		},
		{
			name:    "invalid time zone",
			args:    []string{"ads", "reports", "preset", "--level", "campaigns", "--last-days", "1", "--time-zone", "America/Los_Angeles", "--output", "json"},
			wantErr: "--time-zone must be UTC or ORTZ",
		},
		{
			name:    "search terms require explicit ORTZ",
			args:    []string{"ads", "reports", "preset", "--level", "search-terms", "--campaign", "12345", "--from", recentFrom, "--to", recentTo, "--time-zone", "UTC", "--output", "json"},
			wantErr: "--time-zone must be ORTZ for search-term report levels",
		},
		{
			name:    "last days unsupported for ORTZ",
			args:    []string{"ads", "reports", "preset", "--level", "campaigns", "--last-days", "1", "--time-zone", "ORTZ", "--output", "json"},
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
	if err := root.Parse([]string{"ads", "impression-share-reports", "--limit", "51", "--output", "json"}); err != nil {
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
	if err := root.Parse([]string{"ads", "campaigns", "--limit", "0", "--output", "json"}); err != nil {
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
	if err := root.Parse([]string{"ads", "campaigns", "delete", "--campaign", "123"}); err != nil {
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
		{"ads", "campaigns", "pause", "--campaign", "123", "--confirm", "--output", "json"},
		{"ads", "campaigns", "resume", "--campaign", "123", "--confirm", "--output", "json"},
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
		if stderr != "" {
			t.Fatalf("stderr = %q, want empty", stderr)
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
	if err := root.Parse([]string{"ads", "campaigns", "--org", "123456", "pause", "--campaign", "123", "--confirm"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
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
			args:    []string{"ads", "campaigns", "pause", "--campaign", "123"},
			wantErr: "--confirm is required",
		},
		{
			name:    "invalid campaign",
			args:    []string{"ads", "campaigns", "pause", "--campaign", "abc", "--confirm"},
			wantErr: "--campaign must be an integer",
		},
		{
			name:    "missing campaign",
			args:    []string{"ads", "campaigns", "pause", "--confirm"},
			wantErr: "--campaign is required",
		},
		{
			name:    "parent output conflicts with child pretty",
			args:    []string{"ads", "campaigns", "--output", "table", "pause", "--campaign", "123", "--confirm", "--pretty"},
			wantErr: "--pretty is only valid with JSON output",
		},
		{
			name:    "parent pretty conflicts with child output",
			args:    []string{"ads", "campaigns", "--pretty", "resume", "--campaign", "123", "--confirm", "--output", "table"},
			wantErr: "--pretty is only valid with JSON output",
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
			if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, tc.wantErr) {
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
	if err := root.Parse([]string{"ads", "campaigns", "resume", "--campaign", "123", "--confirm", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if runErr == nil || !strings.Contains(runErr.Error(), "ads campaigns resume:") {
		t.Fatalf("run error = %v, want resume command name", runErr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
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
	if err := root.Parse([]string{"ads", "campaigns", "--output", "json", "unexpected"}); err != nil {
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
	if err := root.Parse([]string{"ads", "api", "request", "--path", "https://example.com/api/v5/campaigns"}); err != nil {
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
	if err := root.Parse([]string{"ads", "api", "request", "--path", "v5/campaigns", "--output", "json", "unexpected"}); err != nil {
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

func adsJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
