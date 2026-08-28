package cmdtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAppsPublishedAuditsEveryAppAndPaginatesTerritories(t *testing.T) {
	setupAuth(t)
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	var appPageTwoReads atomic.Int32
	var appTwoTerritoryReads atomic.Int32
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps" && req.URL.Query().Get("cursor") == "":
			if req.URL.Query().Get("limit") != "200" {
				t.Errorf("apps limit = %q, want 200", req.URL.Query().Get("limit"))
			}
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[
					{"type":"apps","id":"app-2","attributes":{"name":"Zulu","bundleId":"com.example.zulu","sku":"ZULU","primaryLocale":"en-US"}},
					{"type":"apps","id":"app-1","attributes":{"name":"Alpha","bundleId":"com.example.alpha","sku":"ALPHA","primaryLocale":"en-US"}}
				],
				"links":{"next":"https://api.appstoreconnect.apple.com/v1/apps?cursor=Mg"}
			}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps" && req.URL.Query().Get("cursor") == "Mg":
			appPageTwoReads.Add(1)
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[{"type":"apps","id":"app-3","attributes":{"name":"No Availability","bundleId":"com.example.none","sku":"NONE","primaryLocale":"en-US"}}],
				"links":{"next":""}
			}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-1","attributes":{"availableInNewTerritories":false}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-2/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-2","attributes":{"availableInNewTerritories":true}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-3/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusNotFound, `{"errors":[{"status":"404","code":"NOT_FOUND","title":"The specified resource does not exist","detail":"No appAvailabilities resource exists"}]}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-1/territoryAvailabilities" && req.URL.Query().Get("cursor") == "":
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[
					{"type":"territoryAvailabilities","id":"ta-usa","attributes":{"available":false,"contentStatuses":["AVAILABLE"]},"relationships":{"territory":{"data":{"type":"territories","id":"USA"}}}},
					{"type":"territoryAvailabilities","id":"ta-fra","attributes":{"available":true,"contentStatuses":["MISSING_AGREEMENT"]},"relationships":{"territory":{"data":{"type":"territories","id":"FRA"}}}}
				],
				"links":{"next":"https://api.appstoreconnect.apple.com/v2/appAvailabilities/availability-1/territoryAvailabilities?cursor=MQ"}
			}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-1/territoryAvailabilities" && req.URL.Query().Get("cursor") == "MQ":
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[{"type":"territoryAvailabilities","id":"ta-gbr","attributes":{"available":false,"contentStatuses":["AVAILABLE","BETA"]},"relationships":{"territory":{"data":{"type":"territories","id":"GBR"}}}}],
				"links":{"next":""}
			}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-2/territoryAvailabilities":
			appTwoTerritoryReads.Add(1)
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[{"type":"territoryAvailabilities","id":"ta-deu","attributes":{"available":true,"contentStatuses":["PROCESSING"]},"relationships":{"territory":{"data":{"type":"territories","id":"DEU"}}}}],
				"links":{"next":""}
			}`), nil
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"apps", "published", "--pretty"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	var report struct {
		AuditedAppCount   int `json:"auditedAppCount"`
		PublishedAppCount int `json:"publishedAppCount"`
		Apps              []struct {
			ID                      string `json:"id"`
			Name                    string `json:"name"`
			AvailabilityID          string `json:"availabilityId"`
			PublishedTerritoryCount int    `json:"publishedTerritoryCount"`
		} `json:"apps"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode output %q: %v", stdout, err)
	}
	if report.AuditedAppCount != 3 || report.PublishedAppCount != 1 || len(report.Apps) != 1 {
		t.Fatalf("unexpected totals: %+v", report)
	}
	if report.Apps[0].ID != "app-1" || report.Apps[0].Name != "Alpha" || report.Apps[0].AvailabilityID != "availability-1" || report.Apps[0].PublishedTerritoryCount != 2 {
		t.Fatalf("unexpected published app: %+v", report.Apps[0])
	}
	if appPageTwoReads.Load() != 1 || appTwoTerritoryReads.Load() != 1 {
		t.Fatalf("expected complete audit, page2=%d app2-territories=%d", appPageTwoReads.Load(), appTwoTerritoryReads.Load())
	}
	if !strings.Contains(stderr, "Audited 3 app records; found 1 published app") {
		t.Fatalf("unexpected summary: %q", stderr)
	}
}

func TestAppsPublishedEmptyAccount(t *testing.T) {
	setupAuth(t)
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && req.URL.Path == "/v1/apps" {
			return jsonHTTPResponse(http.StatusOK, `{"data":[],"links":{"next":""}}`), nil
		}
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"apps", "published"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stdout != `{"auditedAppCount":0,"publishedAppCount":0,"apps":[]}`+"\n" {
		t.Fatalf("unexpected empty report: %q", stdout)
	}
}

func TestAppsPublishedPreservesSuccessfulRowsWhenAnotherAuditFails(t *testing.T) {
	setupAuth(t)
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	var healthyAppRead atomic.Int32
	http.DefaultTransport = publishedAppsPartialFailureTransport(&healthyAppRead)

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"apps", "published", "--pretty"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected audit error")
		}
		if !strings.Contains(err.Error(), `Broken (bad-app)`) || !strings.Contains(err.Error(), "No apps resource exists") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	var report struct {
		AuditedAppCount   int `json:"auditedAppCount"`
		PublishedAppCount int `json:"publishedAppCount"`
		Apps              []struct {
			ID string `json:"id"`
		} `json:"apps"`
		Failures []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Error string `json:"error"`
		} `json:"failures"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode partial output %q: %v", stdout, err)
	}
	if report.AuditedAppCount != 2 || report.PublishedAppCount != 1 || len(report.Apps) != 1 {
		t.Fatalf("unexpected partial report: %+v", report)
	}
	if report.Apps[0].ID != "good-app" {
		t.Fatalf("unexpected successful app: %+v", report.Apps[0])
	}
	if len(report.Failures) != 1 || report.Failures[0].ID != "bad-app" || report.Failures[0].Name != "Broken" || !strings.Contains(report.Failures[0].Error, "No apps resource exists") {
		t.Fatalf("unexpected failures: %+v", report.Failures)
	}
	if !strings.Contains(stderr, "Audited 2 app records; found 1 published app. 1 app audit(s) failed.") {
		t.Fatalf("unexpected partial summary: %q", stderr)
	}
	if healthyAppRead.Load() != 1 {
		t.Fatalf("healthy app was not audited: %d", healthyAppRead.Load())
	}
}

func TestAppsPublishedRendersPartialResultsForHumanFormats(t *testing.T) {
	setupAuth(t)
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	for _, format := range []string{"table", "markdown"} {
		t.Run(format, func(t *testing.T) {
			http.DefaultTransport = publishedAppsPartialFailureTransport(nil)
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse([]string{"apps", "published", "--output", format}); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				if err := root.Run(context.Background()); err == nil {
					t.Fatal("expected audit error")
				}
			})

			for _, want := range []string{"Healthy", "Failed app audits:", "Broken", "No apps resource exists", "Audited 2 app records; found 1 published app. 1 app audit(s) failed."} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("%s output missing %q: %s", format, want, stdout)
				}
			}
			if strings.Count(stdout, "Audited 2 app records; found 1 published app. 1 app audit(s) failed.") != 1 {
				t.Fatalf("%s output duplicated summary: %s", format, stdout)
			}
			if strings.Count(stdout, "No apps resource exists") != 1 {
				t.Fatalf("%s output duplicated failure detail: %s", format, stdout)
			}
			if stderr != "" {
				t.Fatalf("%s output unexpectedly wrote diagnostics to stderr: %q", format, stderr)
			}
		})
	}
}

func TestAppsPublishedOrdersMultipleFailuresDeterministically(t *testing.T) {
	setupAuth(t)
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps":
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[
					{"type":"apps","id":"z-failure","attributes":{"name":"Zulu Failure","bundleId":"com.example.zulu","sku":"ZULU"}},
					{"type":"apps","id":"healthy","attributes":{"name":"Healthy","bundleId":"com.example.healthy","sku":"HEALTHY"}},
					{"type":"apps","id":"a-failure","attributes":{"name":"Alpha Failure","bundleId":"com.example.alpha","sku":"ALPHA"}}
				],"links":{"next":""}
			}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/healthy/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-healthy","attributes":{}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-healthy/territoryAvailabilities":
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"territoryAvailabilities","id":"territory-healthy","attributes":{"contentStatuses":["AVAILABLE"]}}],"links":{"next":""}}`), nil
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/appAvailabilityV2"):
			return jsonHTTPResponse(http.StatusNotFound, `{"errors":[{"status":"404","code":"NOT_FOUND","title":"The specified resource does not exist","detail":"No apps resource exists"}]}`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"apps", "published", "--pretty"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatal("expected audit error")
	}

	var report struct {
		Failures []struct {
			ID string `json:"id"`
		} `json:"failures"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode partial output %q: %v", stdout, err)
	}
	if len(report.Failures) != 2 || report.Failures[0].ID != "a-failure" || report.Failures[1].ID != "z-failure" {
		t.Fatalf("failures are not deterministically ordered: %+v", report.Failures)
	}
	alphaIndex := strings.Index(runErr.Error(), "Alpha Failure (a-failure)")
	zuluIndex := strings.Index(runErr.Error(), "Zulu Failure (z-failure)")
	if alphaIndex < 0 || zuluIndex < 0 || alphaIndex > zuluIndex {
		t.Fatalf("aggregate error is not deterministically ordered: %v", runErr)
	}
}

func TestAppsPublishedOutputFormatsAndDeterministicOrder(t *testing.T) {
	setupAuth(t)
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = publishedAppsTwoAppTransport(t)

	tests := []struct {
		format string
		want   string
	}{
		{format: "table", want: "Published Territories"},
		{format: "markdown", want: "| ID"},
	}
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse([]string{"apps", "published", "--output", test.format}); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				if err := root.Run(context.Background()); err != nil {
					t.Fatalf("run error: %v", err)
				}
			})
			if !strings.Contains(stdout, test.want) {
				t.Fatalf("%s output missing %q: %s", test.format, test.want, stdout)
			}
			alphaIndex := strings.Index(stdout, "Alpha")
			zuluIndex := strings.Index(stdout, "Zulu")
			if alphaIndex < 0 || zuluIndex < 0 {
				t.Fatalf("%s output missing published apps: %s", test.format, stdout)
			}
			if alphaIndex > zuluIndex {
				t.Fatalf("published apps are not sorted by name: %s", stdout)
			}
			const summary = "Audited 2 app records; found 2 published apps."
			if !strings.Contains(stdout, summary) {
				t.Fatalf("%s output missing audit totals: %s", test.format, stdout)
			}
			if count := strings.Count(stdout+stderr, summary); count != 1 {
				t.Fatalf("%s output contains audit totals %d times, want 1; stdout=%q stderr=%q", test.format, count, stdout, stderr)
			}
		})
	}
}

func TestAppsPublishedRetriesRateLimitedAvailabilityReads(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	var availabilityReads atomic.Int32
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps":
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[{"type":"apps","id":"app-1","attributes":{"name":"Retried","bundleId":"com.example.retried","sku":"RETRIED"}}],
				"links":{"next":""}
			}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/appAvailabilityV2":
			if availabilityReads.Add(1) == 1 {
				return jsonHTTPResponse(http.StatusTooManyRequests, `{"errors":[{"status":"429","code":"RATE_LIMIT_EXCEEDED","title":"Too Many Requests"}]}`), nil
			}
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-1","attributes":{"availableInNewTerritories":false}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-1/territoryAvailabilities":
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"territoryAvailabilities","id":"ta-1","attributes":{"contentStatuses":["AVAILABLE"]}}],"links":{"next":""}}`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"apps", "published"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if availabilityReads.Load() != 2 {
		t.Fatalf("availability reads = %d, want 2", availabilityReads.Load())
	}
	if !strings.Contains(stdout, `"publishedAppCount":1`) {
		t.Fatalf("unexpected output: %s", stdout)
	}
}

func publishedAppsTwoAppTransport(t *testing.T) http.RoundTripper {
	t.Helper()
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps":
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[
					{"type":"apps","id":"z-app","attributes":{"name":"Zulu","bundleId":"com.example.zulu","sku":"ZULU"}},
					{"type":"apps","id":"a-app","attributes":{"name":"Alpha","bundleId":"com.example.alpha","sku":"ALPHA"}}
				],"links":{"next":""}
			}`), nil
		case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/v1/apps/") && strings.HasSuffix(req.URL.Path, "/appAvailabilityV2"):
			appID := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/v1/apps/"), "/appAvailabilityV2")
			return jsonHTTPResponse(http.StatusOK, fmt.Sprintf(`{"data":{"type":"appAvailabilities","id":"availability-%s","attributes":{"availableInNewTerritories":false}}}`, appID)), nil
		case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/v2/appAvailabilities/availability-"):
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"territoryAvailabilities","id":"ta-1","attributes":{"available":true,"contentStatuses":["AVAILABLE"]}}],"links":{"next":""}}`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})
}

func publishedAppsPartialFailureTransport(healthyAppRead *atomic.Int32) http.RoundTripper {
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps":
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[
					{"type":"apps","id":"bad-app","attributes":{"name":"Broken","bundleId":"com.example.broken","sku":"BROKEN"}},
					{"type":"apps","id":"good-app","attributes":{"name":"Healthy","bundleId":"com.example.healthy","sku":"HEALTHY"}}
				],"links":{"next":""}
			}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/bad-app/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusNotFound, `{"errors":[{"status":"404","code":"NOT_FOUND","title":"The specified resource does not exist","detail":"No apps resource exists"}]}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/good-app/appAvailabilityV2":
			if healthyAppRead != nil {
				healthyAppRead.Add(1)
			}
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-good","attributes":{}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-good/territoryAvailabilities":
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"territoryAvailabilities","id":"territory-good","attributes":{"contentStatuses":["AVAILABLE"]}}],"links":{"next":""}}`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})
}
