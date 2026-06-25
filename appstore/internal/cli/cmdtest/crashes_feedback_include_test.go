package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

type betaSubmissionListOutput struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Included []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Version string `json:"version"`
		} `json:"attributes"`
	} `json:"included"`
}

func decodeBetaSubmissionListOutput(t *testing.T, stdout string) betaSubmissionListOutput {
	t.Helper()

	var output betaSubmissionListOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, stdout)
	}
	return output
}

// okJSONResponse builds a 200 JSON response for the mock transport.
func okJSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestCrashesListIncludeBuildSendsBuildRelationship(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	var gotQuery url.Values
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/apps/123/betaFeedbackCrashSubmissions" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		gotQuery = req.URL.Query()
		return okJSONResponse(`{"data":[{"type":"betaFeedbackCrashSubmissions","id":"crash-1"}],` +
			`"included":[{"type":"builds","id":"b1","attributes":{"version":"532621"}}]}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "crashes", "list", "--app", "123", "--include", "build", "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if got := gotQuery.Get("include"); got != "build" {
		t.Fatalf("include query = %q, want %q", got, "build")
	}
	if got := gotQuery.Get("fields[builds]"); got != "version,preReleaseVersion" {
		t.Fatalf("fields[builds] query = %q, want %q", got, "version,preReleaseVersion")
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	output := decodeBetaSubmissionListOutput(t, stdout)
	if len(output.Data) != 1 || output.Data[0].ID != "crash-1" {
		t.Fatalf("unexpected crash data: %+v", output.Data)
	}
	if len(output.Included) != 1 || output.Included[0].Type != "builds" || output.Included[0].ID != "b1" || output.Included[0].Attributes.Version != "532621" {
		t.Fatalf("unexpected included build: %+v", output.Included)
	}
}

func TestCrashesListInvalidIncludeReturnsUsageError(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "crashes", "list", "--app", "123", "--include", "bogus"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if rootcmd.ExitCodeFromError(runErr) != rootcmd.ExitUsage {
		t.Fatalf("exit code = %d, want %d (err=%v)", rootcmd.ExitCodeFromError(runErr), rootcmd.ExitUsage, runErr)
	}
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--include must be a comma-separated list of: build, tester") {
		t.Fatalf("stderr = %q, want --include usage error", stderr)
	}
}

func TestFeedbackListIncludeBuildSendsBuildRelationship(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	var gotQuery url.Values
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/apps/123/betaFeedbackScreenshotSubmissions" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		gotQuery = req.URL.Query()
		return okJSONResponse(`{"data":[{"type":"betaFeedbackScreenshotSubmissions","id":"fb-1"}],` +
			`"included":[{"type":"builds","id":"b1","attributes":{"version":"532621"}}]}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "feedback", "list", "--app", "123", "--include", "build", "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if got := gotQuery.Get("include"); got != "build" {
		t.Fatalf("include query = %q, want %q", got, "build")
	}
	if got := gotQuery.Get("fields[builds]"); got != "version,preReleaseVersion" {
		t.Fatalf("fields[builds] query = %q, want %q", got, "version,preReleaseVersion")
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	output := decodeBetaSubmissionListOutput(t, stdout)
	if len(output.Data) != 1 || output.Data[0].ID != "fb-1" {
		t.Fatalf("unexpected feedback data: %+v", output.Data)
	}
	if len(output.Included) != 1 || output.Included[0].Type != "builds" || output.Included[0].ID != "b1" || output.Included[0].Attributes.Version != "532621" {
		t.Fatalf("unexpected included build: %+v", output.Included)
	}
}

func TestFeedbackListInvalidIncludeReturnsUsageError(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "feedback", "list", "--app", "123", "--include", "bogus"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if rootcmd.ExitCodeFromError(runErr) != rootcmd.ExitUsage {
		t.Fatalf("exit code = %d, want %d (err=%v)", rootcmd.ExitCodeFromError(runErr), rootcmd.ExitUsage, runErr)
	}
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--include must be a comma-separated list of: build, tester") {
		t.Fatalf("stderr = %q, want --include usage error", stderr)
	}
}
