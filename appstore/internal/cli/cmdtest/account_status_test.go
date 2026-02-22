package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccountStatusJSON(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/apps" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		if req.URL.Query().Get("limit") != "1" {
			t.Fatalf("expected apps list limit=1, got %q", req.URL.Query().Get("limit"))
		}
		return insightsJSONResponse(`{
			"data":[
				{"type":"apps","id":"app-1","attributes":{"name":"My App","bundleId":"com.example.myapp","sku":"sku"}}
			],
			"links":{"next":""}
		}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"account", "status"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("unmarshal output: %v\nstdout=%s", err, stdout)
	}

	summary, ok := payload["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got %T", payload["summary"])
	}
	if summary["health"] == "" {
		t.Fatalf("expected summary.health, got %v", summary)
	}

	checks, ok := payload["checks"].([]any)
	if !ok || len(checks) == 0 {
		t.Fatalf("expected checks array, got %T %v", payload["checks"], payload["checks"])
	}

	foundAPICheck := false
	for _, item := range checks {
		check, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if check["name"] == "api_access" {
			foundAPICheck = true
		}
	}
	if !foundAPICheck {
		t.Fatalf("expected api_access check in %v", checks)
	}
}

func TestAccountStatusForbiddenProducesWarning(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/apps" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body: io.NopCloser(strings.NewReader(`{
				"errors":[{"status":"403","code":"FORBIDDEN","title":"Forbidden","detail":"permission denied"}]
			}`)),
			Header: http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"account", "status"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("unmarshal output: %v\nstdout=%s", err, stdout)
	}

	checks, ok := payload["checks"].([]any)
	if !ok {
		t.Fatalf("expected checks array, got %T", payload["checks"])
	}
	var apiStatus string
	for _, item := range checks {
		check, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if check["name"] == "api_access" {
			apiStatus, _ = check["status"].(string)
		}
	}
	if apiStatus != "warn" {
		t.Fatalf("expected api_access warn status, got %q", apiStatus)
	}
}

func TestAccountStatusTableOutput(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return insightsJSONResponse(`{
			"data":[
				{"type":"apps","id":"app-1","attributes":{"name":"My App","bundleId":"com.example.myapp","sku":"sku"}}
			],
			"links":{"next":""}
		}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"account", "status", "--output", "table"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "SUMMARY") || !strings.Contains(stdout, "CHECKS") {
		t.Fatalf("expected summary/checks sections in table output, got %q", stdout)
	}
}
