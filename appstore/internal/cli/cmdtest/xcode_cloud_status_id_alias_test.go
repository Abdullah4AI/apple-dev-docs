package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestXcodeCloudStatusDeprecatedIDAliasReturnsJSON(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if req.Method != http.MethodGet || req.URL.Path != "/v1/ciBuildRuns/run-1" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		body := `{"data":{"type":"ciBuildRuns","id":"run-1","attributes":{"number":42,"executionProgress":"COMPLETE","completionStatus":"SUCCEEDED"}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"xcode-cloud", "status", "--id", "run-1", "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	var result struct {
		BuildRunID string `json:"buildRunId"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode output %q: %v", stdout, err)
	}
	if result.BuildRunID != "run-1" {
		t.Fatalf("buildRunId = %q, want run-1", result.BuildRunID)
	}
	if stderr != "Warning: `--id` is deprecated. Use `--run-id`.\n" {
		t.Fatalf("stderr = %q, want deprecation warning", stderr)
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
}

func TestXcodeCloudStatusRejectsIDAndRunIDBeforeNetwork(t *testing.T) {
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, errors.New("unexpected network request")
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"xcode-cloud", "status", "--run-id", "run-1", "--id", "run-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !shared.IsReportedUsageError(runErr) {
		t.Fatalf("run error = %v, want concise usage error", runErr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "--id conflicts with --run-id; use only --run-id") {
		t.Fatalf("stderr = %q, want dual-use conflict", stderr)
	}
	if strings.Contains(stderr, "DESCRIPTION\n") || strings.Contains(stderr, "USAGE\n") {
		t.Fatalf("stderr includes full help: %q", stderr)
	}
	diagnostic, ok := shared.DiagnosticFromError(runErr)
	if !ok || diagnostic.Code != shared.DiagnosticConflictingInput || diagnostic.Parameter != "--run-id" {
		t.Fatalf("diagnostic = %+v (ok=%t), want canonical --run-id conflict", diagnostic, ok)
	}
}
