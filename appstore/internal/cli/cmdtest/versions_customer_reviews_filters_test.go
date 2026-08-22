package cmdtest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

const versionCustomerReviewsPayload = `{"data":[{"type":"customerReviews","id":"review-1","attributes":{"rating":2,"title":"Slow","body":"Laggy","reviewerNickname":"Tester","createdDate":"2026-01-20T00:00:00Z","territory":"USA"}}]}`

func TestVersionsCustomerReviewsListEmitsReviewFilters(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/v1/appStoreVersions/version-1/customerReviews" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		query := req.URL.Query()
		for _, want := range []struct {
			key   string
			value string
		}{
			{key: "filter[rating]", value: "1,2"},
			{key: "filter[territory]", value: "USA,GBR"},
			{key: "sort", value: "-createdDate"},
			{key: "exists[publishedResponse]", value: "false"},
			{key: "include", value: "response"},
			{key: "fields[customerReviewResponses]", value: "responseBody,state"},
			{key: "limit", value: "5"},
		} {
			if got := query.Get(want.key); got != want.value {
				t.Errorf("%s = %q, want %q (raw query %q)", want.key, got, want.value, req.URL.RawQuery)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, versionCustomerReviewsPayload)
	}))
	t.Cleanup(server.Close)

	client := newReviewTestServerClient(t, server)
	restoreClient := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) { return client, nil })
	t.Cleanup(restoreClient)

	stdout, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{
			"versions", "customer-reviews", "list",
			"--version-id", "version-1",
			"--stars", "1,2",
			"--territory", "us,GB",
			"--sort", "-createdDate",
			"--response-state", "unreplied",
			"--response-fields", "responseBody,state",
			"--limit", "5",
			"--output", "json",
		}, "1.2.3")
		if code != rootcmd.ExitSuccess {
			t.Fatalf("exit code = %d, want %d", code, rootcmd.ExitSuccess)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"review-1"`) {
		t.Fatalf("expected review envelope on stdout, got %q", stdout)
	}
	if requests != 1 {
		t.Fatalf("request count = %d, want 1", requests)
	}
}

func TestVersionsCustomerReviewsListOnlyUnrespondedAndIncludeResponse(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		if got := query.Get("exists[publishedResponse]"); got != "false" {
			t.Errorf("exists[publishedResponse] = %q, want false", got)
		}
		if got := query.Get("include"); got != "response" {
			t.Errorf("include = %q, want response", got)
		}
		if _, ok := query["fields[customerReviewResponses]"]; ok {
			t.Errorf("fields[customerReviewResponses] must be absent, got %q", req.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, versionCustomerReviewsPayload)
	}))
	t.Cleanup(server.Close)

	client := newReviewTestServerClient(t, server)
	restoreClient := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) { return client, nil })
	t.Cleanup(restoreClient)

	_, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{
			"versions", "customer-reviews", "list",
			"--version-id", "version-1",
			"--only-unresponded",
			"--include-response",
			"--output", "json",
		}, "1.2.3")
		if code != rootcmd.ExitSuccess {
			t.Fatalf("exit code = %d, want %d", code, rootcmd.ExitSuccess)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}

func TestVersionsCustomerReviewsListRejectsInvalidStars(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, versionCustomerReviewsPayload)
	}))
	t.Cleanup(server.Close)

	client := newReviewTestServerClient(t, server)
	restoreClient := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) { return client, nil })
	t.Cleanup(restoreClient)

	stdout, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{
			"versions", "customer-reviews", "list",
			"--version-id", "version-1",
			"--stars", "9",
			"--output", "json",
		}, "1.2.3")
		if code != rootcmd.ExitUsage {
			t.Fatalf("exit code = %d, want %d", code, rootcmd.ExitUsage)
		}
	})
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--stars must be a comma-separated list of star ratings: 1, 2, 3, 4, 5") {
		t.Fatalf("expected --stars guidance listing valid values, got %q", stderr)
	}
	if requests != 0 {
		t.Fatalf("request count = %d, want 0 (validation must precede the request)", requests)
	}
}

// The version-scoped listing shares Apple's customer review filter surface with
// the app-level listing, so identical bad input must fail identically on both.
func TestVersionsCustomerReviewsFilterValidationMatchesAppLevelReviews(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{
			name:        "explicitly empty star rating",
			args:        []string{"--stars", ""},
			wantMessage: "--stars must be a comma-separated list of star ratings: 1, 2, 3, 4, 5",
		},
		{
			name:        "explicitly empty territory",
			args:        []string{"--territory", ""},
			wantMessage: "--territory must be a valid App Store territory code",
		},
		{
			name:        "explicitly empty sort",
			args:        []string{"--sort", ""},
			wantMessage: "--sort must be one of: rating, -rating, createdDate, -createdDate",
		},
		{
			name:        "explicitly empty response state",
			args:        []string{"--response-state", ""},
			wantMessage: "--response-state must be one of: any, unresponded, unreplied, responded, replied",
		},
		{
			name:        "whitespace response fields",
			args:        []string{"--response-fields", "   "},
			wantMessage: "--response-fields must be a comma-separated list of: responseBody,lastModifiedDate,state,review",
		},
		{
			name:        "star rating out of range",
			args:        []string{"--stars", "9"},
			wantMessage: "--stars must be a comma-separated list of star ratings: 1, 2, 3, 4, 5",
		},
		{
			name:        "star rating not numeric",
			args:        []string{"--stars", "five"},
			wantMessage: "--stars must be a comma-separated list of star ratings: 1, 2, 3, 4, 5",
		},
		{
			name:        "unsupported sort",
			args:        []string{"--sort", "bogus"},
			wantMessage: "--sort must be one of: rating, -rating, createdDate, -createdDate",
		},
		{
			name:        "unsupported response state",
			args:        []string{"--response-state", "maybe"},
			wantMessage: "--response-state must be one of: any, unresponded, unreplied, responded, replied",
		},
		{
			name:        "unsupported response fields",
			args:        []string{"--response-fields", "bogus"},
			wantMessage: "--response-fields must be a comma-separated list of: responseBody,lastModifiedDate,state,review",
		},
		{
			name:        "unsupported territory",
			args:        []string{"--territory", "not-a-territory"},
			wantMessage: "--territory must be a valid App Store territory code",
		},
		{
			name:        "response fields with a repeated comma",
			args:        []string{"--response-fields", "responseBody,,state"},
			wantMessage: "--response-fields must be a comma-separated list of: responseBody,lastModifiedDate,state,review",
		},
		{
			name:        "response fields with no usable elements",
			args:        []string{"--response-fields", ","},
			wantMessage: "--response-fields must be a comma-separated list of: responseBody,lastModifiedDate,state,review",
		},
		{
			name:        "conflicting response filters",
			args:        []string{"--only-unresponded", "--response-state", "responded"},
			wantMessage: "--only-unresponded cannot be combined with --response-state responded",
		},
		{
			name:        "repeated star filters",
			args:        []string{"--stars", "1", "--stars", "2"},
			wantMessage: `--stars specified multiple times; pass one comma-separated list, for example --stars "1,2"`,
		},
		{
			name:        "repeated territory filters",
			args:        []string{"--territory", "USA", "--territory", "GBR"},
			wantMessage: `--territory specified multiple times; pass one comma-separated list, for example --territory "USA,GBR"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			appCode, appStderr := runReviewFilterFailure(t, append([]string{"reviews", "list", "--app", "app-1"}, test.args...))
			versionCode, versionStderr := runReviewFilterFailure(t, append([]string{"versions", "customer-reviews", "list", "--version-id", "version-1"}, test.args...))

			if appCode == rootcmd.ExitSuccess {
				t.Fatalf("app-level reviews list unexpectedly succeeded")
			}
			if versionCode != appCode {
				t.Fatalf("version-scoped exit code = %d, app-level exit code = %d; they must match", versionCode, appCode)
			}
			if !strings.Contains(appStderr, test.wantMessage) {
				t.Fatalf("app-level stderr = %q, want it to contain %q", appStderr, test.wantMessage)
			}
			if !strings.Contains(versionStderr, test.wantMessage) {
				t.Fatalf("version-scoped stderr = %q, want it to contain %q", versionStderr, test.wantMessage)
			}
		})
	}
}

func TestCustomerReviewListsRejectQueryFlagsWithNext(t *testing.T) {
	const nextURL = "https://api.appstoreconnect.apple.com/v1/customerReviews?cursor=next"
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "version stars",
			args:    []string{"versions", "customer-reviews", "list", "--next", nextURL, "--stars", "1"},
			wantErr: "--next cannot be combined with --stars",
		},
		{
			name:    "app response fields",
			args:    []string{"reviews", "list", "--next", nextURL, "--response-fields", "responseBody"},
			wantErr: "--next cannot be combined with --response-fields",
		},
		{
			name:    "explicit empty territory",
			args:    []string{"versions", "customer-reviews", "list", "--next", nextURL, "--territory", ""},
			wantErr: "--next cannot be combined with --territory",
		},
		{
			name:    "explicit zero limit",
			args:    []string{"reviews", "--next", nextURL, "--limit", "0"},
			wantErr: "--next cannot be combined with --limit",
		},
		{
			name:    "version owner",
			args:    []string{"versions", "customer-reviews", "list", "--next", nextURL, "--version-id", "version-1"},
			wantErr: "--next cannot be combined with --version-id",
		},
		{
			name:    "app owner",
			args:    []string{"reviews", "list", "--next", nextURL, "--app", "app-1"},
			wantErr: "--next cannot be combined with --app",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, stderr := runReviewFilterFailure(t, test.args)
			if code != rootcmd.ExitUsage {
				t.Fatalf("exit code = %d, want %d; stderr=%q", code, rootcmd.ExitUsage, stderr)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("stderr = %q, want it to contain %q", stderr, test.wantErr)
			}
		})
	}
}

func runReviewFilterFailure(t *testing.T, args []string) (int, string) {
	t.Helper()

	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		t.Errorf("unexpected request %s %s: validation must precede the request", req.Method, req.URL.String())
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, versionCustomerReviewsPayload)
	}))
	t.Cleanup(server.Close)

	client := newReviewTestServerClient(t, server)
	restoreClient := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) { return client, nil })
	t.Cleanup(restoreClient)

	code := 0
	_, stderr := captureOutput(t, func() {
		code = rootcmd.Run(append(args, "--output", "json"), "1.2.3")
	})
	return code, stderr
}
