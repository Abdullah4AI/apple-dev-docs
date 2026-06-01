package cmdtest

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSelectorLatestExcludeExpiredFiltersResolvedBuild(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if req.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", req.Method)
			}
			if req.URL.Path != "/v1/builds" {
				t.Fatalf("expected builds lookup, got %s", req.URL.Path)
			}
			query := req.URL.Query()
			if got := query.Get("filter[app]"); got != "123456789" {
				t.Fatalf("expected app filter 123456789, got %q", got)
			}
			if got := query.Get("filter[expired]"); got != "false" {
				t.Fatalf("expected expired=false filter, got %q", got)
			}
			if got := query.Get("sort"); got != "-uploadedDate" {
				t.Fatalf("expected sort=-uploadedDate, got %q", got)
			}
			body := `{"data":[{"type":"builds","id":"build-active","attributes":{"version":"42","uploadedDate":"2026-05-31T00:00:00Z"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 2:
			if req.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", req.Method)
			}
			if req.URL.Path != "/v1/builds/build-active/app" {
				t.Fatalf("expected build app lookup, got %s", req.URL.Path)
			}
			body := `{"data":{"type":"apps","id":"123456789","attributes":{"name":"Example"}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected request %d: %s %s", requestCount, req.Method, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"builds", "app", "view", "--app", "123456789", "--latest", "--exclude-expired"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"123456789"`) {
		t.Fatalf("expected app output, got %q", stdout)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
}
