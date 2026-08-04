package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const analyticsViewRequestID = "11111111-1111-1111-1111-111111111111"

func TestAnalyticsViewDeprecatedDatePreservesReportDateMatching(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if req.Method != http.MethodGet || req.URL.Path != "/v1/analyticsReportRequests/"+analyticsViewRequestID+"/reports" {
				t.Fatalf("unexpected reports request: %s %s", req.Method, req.URL.String())
			}
			return analyticsViewJSONResponse(`{
				"data":[{
					"type":"analyticsReports",
					"id":"report-1",
					"attributes":{"name":"App Sessions","reportType":"STANDARD","category":"APP_USAGE","granularity":"DAILY"}
				}],
				"links":{"self":"https://api.appstoreconnect.apple.com/v1/analyticsReportRequests/11111111-1111-1111-1111-111111111111/reports"}
			}`), nil
		case 2:
			if req.Method != http.MethodGet || req.URL.Path != "/v1/analyticsReports/report-1/instances" {
				t.Fatalf("unexpected instances request: %s %s", req.Method, req.URL.String())
			}
			if req.URL.Query().Has("filter[processingDate]") {
				t.Fatalf("deprecated --date must not become a processing-date API filter: %s", req.URL.String())
			}
			return analyticsViewJSONResponse(`{
				"data":[{
					"type":"analyticsReportInstances",
					"id":"instance-1",
					"attributes":{"reportDate":"2024-01-20","processingDate":"2024-01-21","granularity":"DAILY","version":"1.0"}
				}],
				"links":{}
			}`), nil
		default:
			t.Fatalf("unexpected extra request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	stdout, stderr := runAnalyticsViewForOutput(
		t,
		"--request-id", analyticsViewRequestID,
		"--date", "2024-01-20",
		"--output", "json",
	)
	const warning = "Warning: `--date` is deprecated. Use `--processing-date`.\n"
	if stderr != warning {
		t.Fatalf("stderr = %q, want %q", stderr, warning)
	}
	assertAnalyticsViewJSONEqual(t, stdout, `{
		"requestId":"11111111-1111-1111-1111-111111111111",
		"data":[{
			"id":"report-1",
			"reportType":"STANDARD",
			"name":"App Sessions",
			"category":"APP_USAGE",
			"granularity":"DAILY",
			"instances":[{
				"id":"instance-1",
				"reportDate":"2024-01-20",
				"processingDate":"2024-01-21",
				"granularity":"DAILY",
				"version":"1.0"
			}]
		}],
		"links":{"self":"https://api.appstoreconnect.apple.com/v1/analyticsReportRequests/11111111-1111-1111-1111-111111111111/reports"}
	}`)
	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
}

func TestAnalyticsViewProcessingDateForwardsFilter(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			return analyticsViewJSONResponse(`{
				"data":[{"type":"analyticsReports","id":"report-1","attributes":{"name":"App Sessions","granularity":"DAILY"}}],
				"links":{}
			}`), nil
		case 2:
			if got := req.URL.Query().Get("filter[processingDate]"); got != "2024-01-20" {
				t.Fatalf("filter[processingDate] = %q, want %q", got, "2024-01-20")
			}
			return analyticsViewJSONResponse(`{
				"data":[{"type":"analyticsReportInstances","id":"instance-1","attributes":{"processingDate":"2024-01-20","granularity":"DAILY"}}],
				"links":{}
			}`), nil
		default:
			t.Fatalf("unexpected extra request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	_, stderr := runAnalyticsViewForOutput(
		t,
		"--request-id", analyticsViewRequestID,
		"--processing-date", "2024-01-20",
		"--output", "json",
	)
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestAnalyticsViewGranularityOnlyPreservesReportMetadata(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			return analyticsViewJSONResponse(`{
				"data":[{"type":"analyticsReports","id":"report-1","attributes":{"name":"Sales","reportType":"STANDARD","category":"COMMERCE","granularity":"MONTHLY"}}],
				"links":{}
			}`), nil
		case 2:
			query := req.URL.Query()
			if got := query.Get("filter[granularity]"); got != "MONTHLY" {
				t.Fatalf("filter[granularity] = %q, want MONTHLY", got)
			}
			if query.Has("filter[processingDate]") {
				t.Fatalf("unexpected processing-date filter: %s", req.URL.String())
			}
			return analyticsViewJSONResponse(`{"data":[],"links":{}}`), nil
		default:
			t.Fatalf("unexpected extra request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	stdout, stderr := runAnalyticsViewForOutput(
		t,
		"--request-id", analyticsViewRequestID,
		"--granularity", "monthly",
		"--output", "json",
	)
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	assertAnalyticsViewJSONEqual(t, stdout, `{
		"requestId":"11111111-1111-1111-1111-111111111111",
		"data":[{
			"id":"report-1",
			"reportType":"STANDARD",
			"name":"Sales",
			"category":"COMMERCE",
			"granularity":"MONTHLY"
		}],
		"links":{}
	}`)
	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
}

func TestAnalyticsViewFiltersPaginateAndPreserveOutput(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	const (
		reportsNextURL   = "https://api.appstoreconnect.apple.com/v1/analyticsReportRequests/11111111-1111-1111-1111-111111111111/reports?cursor=reports-next"
		instancesNextURL = "https://api.appstoreconnect.apple.com/v1/analyticsReports/report-1/instances?cursor=instances-next"
	)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if req.URL.Path != "/v1/analyticsReportRequests/"+analyticsViewRequestID+"/reports" || req.URL.Query().Get("limit") != "200" {
				t.Fatalf("unexpected first reports request: %s", req.URL.String())
			}
			return analyticsViewJSONResponse(`{
				"data":[{"type":"analyticsReports","id":"report-1","attributes":{"name":"App Sessions","reportType":"STANDARD","category":"APP_USAGE","granularity":"DAILY"}}],
				"links":{"self":"https://api.appstoreconnect.apple.com/v1/analyticsReportRequests/11111111-1111-1111-1111-111111111111/reports","next":"` + reportsNextURL + `"}
			}`), nil
		case 2:
			if req.URL.String() != reportsNextURL {
				t.Fatalf("reports next URL = %q, want %q", req.URL.String(), reportsNextURL)
			}
			return analyticsViewJSONResponse(`{
				"data":[{"type":"analyticsReports","id":"report-2","attributes":{"name":"Store Discovery","reportType":"STANDARD","category":"APP_STORE_ENGAGEMENT","granularity":"WEEKLY"}}],
				"links":{}
			}`), nil
		case 3:
			assertAnalyticsInstanceFilters(t, req, "/v1/analyticsReports/report-1/instances")
			return analyticsViewJSONResponse(`{
				"data":[{"type":"analyticsReportInstances","id":"instance-1","attributes":{"reportDate":"2024-01-19","processingDate":"2024-01-20","granularity":"DAILY","version":"1.0"}}],
				"links":{"next":"` + instancesNextURL + `"}
			}`), nil
		case 4:
			if req.URL.String() != instancesNextURL {
				t.Fatalf("instances next URL = %q, want %q", req.URL.String(), instancesNextURL)
			}
			return analyticsViewJSONResponse(`{
				"data":[{"type":"analyticsReportInstances","id":"instance-2","attributes":{"reportDate":"2024-01-19","processingDate":"2024-01-20","granularity":"WEEKLY","version":"1.0"}}],
				"links":{}
			}`), nil
		case 5:
			assertAnalyticsInstanceFilters(t, req, "/v1/analyticsReports/report-2/instances")
			return analyticsViewJSONResponse(`{"data":[],"links":{}}`), nil
		default:
			t.Fatalf("unexpected extra request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	stdout, stderr := runAnalyticsViewForOutput(
		t,
		"--request-id", analyticsViewRequestID,
		"--processing-date", "2024-01-20",
		"--granularity", "daily, weekly, monthly",
		"--paginate",
		"--output", "json",
	)
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	assertAnalyticsViewJSONEqual(t, stdout, `{
		"requestId":"11111111-1111-1111-1111-111111111111",
		"data":[{
			"id":"report-1",
			"reportType":"STANDARD",
			"name":"App Sessions",
			"category":"APP_USAGE",
			"granularity":"DAILY",
			"instances":[
				{"id":"instance-1","reportDate":"2024-01-19","processingDate":"2024-01-20","granularity":"DAILY","version":"1.0"},
				{"id":"instance-2","reportDate":"2024-01-19","processingDate":"2024-01-20","granularity":"WEEKLY","version":"1.0"}
			]
		}],
		"links":{"self":"https://api.appstoreconnect.apple.com/v1/analyticsReportRequests/11111111-1111-1111-1111-111111111111/reports"}
	}`)
	if requestCount != 5 {
		t.Fatalf("expected 5 requests, got %d", requestCount)
	}
}

func TestAnalyticsViewDeprecatedDateConflictsWithProcessingDate(t *testing.T) {
	stdout, stderr, err := runCommand(t, []string{
		"analytics", "view",
		"--request-id", analyticsViewRequestID,
		"--processing-date", "2024-01-20",
		"--date", "2024-01-21",
	})
	if err == nil {
		t.Fatal("expected conflicting date flags to fail")
	}
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v, want usage error", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	const warning = "Warning: `--date` is deprecated. Use `--processing-date`."
	if !strings.Contains(stderr, warning) {
		t.Fatalf("stderr = %q, want warning %q", stderr, warning)
	}
	const conflict = "Error: --date conflicts with --processing-date; use only --processing-date"
	if !strings.Contains(stderr, conflict) {
		t.Fatalf("stderr = %q, want conflict %q", stderr, conflict)
	}
}

func TestAnalyticsViewNoFiltersPreservesInstanceAndSegmentBehavior(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	const instanceID = "22222222-2222-2222-2222-222222222222"
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if req.URL.Path != "/v1/analyticsReportRequests/"+analyticsViewRequestID+"/reports" || req.URL.RawQuery != "limit=200" {
				t.Fatalf("unexpected reports request: %s", req.URL.String())
			}
			return analyticsViewJSONResponse(`{
				"data":[{"type":"analyticsReports","id":"report-1","attributes":{"name":"App Sessions","granularity":"DAILY"}}],
				"links":{}
			}`), nil
		case 2:
			if req.URL.Path != "/v1/analyticsReports/report-1/instances" || req.URL.RawQuery != "limit=200" {
				t.Fatalf("unexpected unfiltered instances request: %s", req.URL.String())
			}
			return analyticsViewJSONResponse(`{
				"data":[
					{"type":"analyticsReportInstances","id":"ignored-instance","attributes":{"processingDate":"2024-01-19"}},
					{"type":"analyticsReportInstances","id":"` + instanceID + `","attributes":{"reportDate":"2024-01-19","processingDate":"2024-01-20","granularity":"DAILY","version":"1.0"}}
				],
				"links":{}
			}`), nil
		case 3:
			if req.URL.Path != "/v1/analyticsReportInstances/"+instanceID+"/segments" || req.URL.RawQuery != "limit=200" {
				t.Fatalf("unexpected segments request: %s", req.URL.String())
			}
			return analyticsViewJSONResponse(`{
				"data":[{"type":"analyticsReportSegments","id":"segment-1","attributes":{"url":"https://example.com/report.gz","checksum":"abc123","sizeInBytes":42,"urlExpirationDate":"2024-01-21T00:00:00Z"}}],
				"links":{}
			}`), nil
		default:
			t.Fatalf("unexpected extra request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	stdout, stderr := runAnalyticsViewForOutput(
		t,
		"--request-id", analyticsViewRequestID,
		"--instance-id", instanceID,
		"--include-segments",
		"--output", "json",
	)
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	assertAnalyticsViewJSONEqual(t, stdout, `{
		"requestId":"11111111-1111-1111-1111-111111111111",
		"data":[{
			"id":"report-1",
			"name":"App Sessions",
			"granularity":"DAILY",
			"instances":[{
				"id":"22222222-2222-2222-2222-222222222222",
				"reportDate":"2024-01-19",
				"processingDate":"2024-01-20",
				"granularity":"DAILY",
				"version":"1.0",
				"segments":[{
					"id":"segment-1",
					"downloadUrl":"https://example.com/report.gz",
					"checksum":"abc123",
					"sizeInBytes":42,
					"urlExpirationDate":"2024-01-21T00:00:00Z"
				}]
			}]
		}],
		"links":{}
	}`)
	if requestCount != 3 {
		t.Fatalf("expected 3 requests, got %d", requestCount)
	}
}

func assertAnalyticsInstanceFilters(t *testing.T, req *http.Request, wantPath string) {
	t.Helper()
	if req.Method != http.MethodGet || req.URL.Path != wantPath {
		t.Fatalf("unexpected instances request: %s %s", req.Method, req.URL.String())
	}
	query := req.URL.Query()
	if got := query.Get("filter[processingDate]"); got != "2024-01-20" {
		t.Fatalf("filter[processingDate] = %q, want %q", got, "2024-01-20")
	}
	if got := query.Get("filter[granularity]"); got != "DAILY,WEEKLY,MONTHLY" {
		t.Fatalf("filter[granularity] = %q, want %q", got, "DAILY,WEEKLY,MONTHLY")
	}
	if got := query.Get("limit"); got != "200" {
		t.Fatalf("limit = %q, want %q", got, "200")
	}
}

func runAnalyticsViewForOutput(t *testing.T, args ...string) (string, string) {
	t.Helper()
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	fullArgs := append([]string{"analytics", "view"}, args...)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse(fullArgs); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	return stdout, stderr
}

func assertAnalyticsViewJSONEqual(t *testing.T, gotJSON, wantJSON string) {
	t.Helper()
	var got any
	if err := json.Unmarshal([]byte(gotJSON), &got); err != nil {
		t.Fatalf("failed to parse command output: %v\noutput=%s", err, gotJSON)
	}
	var want any
	if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
		t.Fatalf("failed to parse expected output: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected JSON output\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}

func analyticsViewJSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}
