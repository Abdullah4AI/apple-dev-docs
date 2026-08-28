package cmdtest

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestFlightMetricsAppTestersSinglePageWarnsOnMorePages(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	keyPath := filepath.Join(t.TempDir(), "AuthKey.p8")
	writeECDSAPEM(t, keyPath)
	t.Setenv("ASC_KEY_ID", "TEST_KEY")
	t.Setenv("ASC_ISSUER_ID", "TEST_ISSUER")
	t.Setenv("ASC_PRIVATE_KEY_PATH", keyPath)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"data":[{"id":"usage-1"}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/apps/app-123/metrics/betaTesterUsages?page=2"}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "metrics", "app-testers", "--app", "app-123"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if !strings.Contains(stdout, `"usage-1"`) {
		t.Fatalf("expected metric data in stdout, got %q", stdout)
	}
	wantWarning := "Warning: showing 1 results; more pages exist (use --paginate or --next where supported)\n"
	if stderr != wantWarning {
		t.Fatalf("stderr = %q, want %q", stderr, wantWarning)
	}
}

func TestTestFlightMetricsGroupTestersSinglePageWarnsWithPagingTotal(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/betaGroups/group-1/metrics/betaTesterUsages" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		if req.URL.Query().Get("groupBy") != "betaTesters" {
			t.Fatalf("expected groupBy=betaTesters, got %q", req.URL.Query().Get("groupBy"))
		}
		body := `{"data":[{"type":"appsBetaTesterUsages","dataPoints":[{"start":"2026-08-01T00:00:00Z","end":"2026-08-02T00:00:00Z","values":{"sessionCount":12}}],"dimensions":{"betaTesters":{"data":{"type":"betaTesters","id":"tester-1"}}}}],"links":{"next":"https://api.example.test/v1/betaGroups/group-1/metrics/betaTesterUsages?cursor=next"},"meta":{"paging":{"total":7,"limit":1}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "metrics", "group-testers", "--group", "group-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if !strings.Contains(stdout, `"tester-1"`) {
		t.Fatalf("expected metric data in stdout, got %q", stdout)
	}
	wantWarning := "Warning: showing 1 of 7 results; more pages exist (use --paginate or --next where supported)\n"
	if stderr != wantWarning {
		t.Fatalf("stderr = %q, want %q", stderr, wantWarning)
	}
}

func TestTestFlightMetricsResolveTestersMergesIncludedAcrossPages(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	keyPath := filepath.Join(t.TempDir(), "AuthKey.p8")
	writeECDSAPEM(t, keyPath)
	t.Setenv("ASC_KEY_ID", "TEST_KEY")
	t.Setenv("ASC_ISSUER_ID", "TEST_ISSUER")
	t.Setenv("ASC_PRIVATE_KEY_PATH", keyPath)

	secondURL := "https://api.appstoreconnect.apple.com/v1/apps/app-123/metrics/betaTesterUsages?page=2"

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	callCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		switch callCount {
		case 1:
			body := `{"data":[{"dataPoints":[{"start":"2026-05-20","end":"2026-08-18","values":{"sessionCount":3}}],"dimensions":{"betaTesters":{"data":"tester-1"}}}],"included":[{"type":"betaTesters","id":"tester-1","attributes":{"firstName":"Ada","lastName":"Lovelace","email":"ada@example.com","state":"INSTALLED"}}],"links":{"next":"` + secondURL + `"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 2:
			body := `{"data":[{"dataPoints":[{"start":"2026-05-20","end":"2026-08-18","values":{"sessionCount":9}}],"dimensions":{"betaTesters":{"data":"tester-2"}}}],"included":[{"type":"betaTesters","id":"tester-2","attributes":{"firstName":"Grace","lastName":"Hopper","email":"grace@example.com","state":"INVITED"}}],"links":{"next":""}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Errorf("unexpected request %d to %s — included resources should satisfy resolution without tester fetches", callCount, req.URL.String())
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "metrics", "app-testers", "--app", "app-123", "--paginate", "--resolve-testers"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if callCount != 2 {
		t.Fatalf("expected exactly 2 metric page fetches, got %d", callCount)
	}
	for _, want := range []string{`"ada@example.com"`, `"grace@example.com"`, `"tester-1"`, `"tester-2"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %s in resolved output, got %q", want, stdout)
		}
	}
}
