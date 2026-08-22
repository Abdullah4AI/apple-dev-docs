package cmdtest

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

func TestLocalizationsListSendsIncludeAndPrintsEnvelope(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if got := req.URL.Path; got != "/v1/appStoreVersions/version-1/appStoreVersionLocalizations" {
			t.Fatalf("unexpected path: %s", got)
		}
		if got := req.URL.Query().Get("include"); got != "appScreenshotSets,appPreviewSets" {
			t.Fatalf("expected include=appScreenshotSets,appPreviewSets, got %q", got)
		}
		if got := req.URL.Query().Get("filter[locale]"); got != "en-US" {
			t.Fatalf("expected filter[locale]=en-US, got %q", got)
		}
		body := `{"data":[{"type":"appStoreVersionLocalizations","id":"loc-1","attributes":{"locale":"en-US"}}],` +
			`"included":[{"type":"appScreenshotSets","id":"set-1"}],"links":{"self":"https://api.appstoreconnect.apple.com/v1/appStoreVersions/version-1/appStoreVersionLocalizations"}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"localizations", "list",
			"--version", "version-1",
			"--locale", "en-US",
			"--include", "appScreenshotSets,appPreviewSets",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if requestCount != 1 {
		t.Fatalf("expected exactly 1 request, got %d", requestCount)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"loc-1"`) {
		t.Fatalf("expected localization data in output, got %q", stdout)
	}
	if !strings.Contains(stdout, `"included":[{"type":"appScreenshotSets","id":"set-1"}]`) {
		t.Fatalf("expected unmodified included envelope in output, got %q", stdout)
	}
}

func TestLocalizationsListPaginateMergesIncludeAcrossPages(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	const secondURL = "https://api.appstoreconnect.apple.com/v1/appStoreVersions/version-1/appStoreVersionLocalizations?cursor=BQ&include=appScreenshotSets&limit=200"

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if got := req.URL.Query().Get("include"); got != "appScreenshotSets" {
				t.Fatalf("expected include=appScreenshotSets on first page, got %q", got)
			}
			if got := req.URL.Query().Get("limit"); got != "200" {
				t.Fatalf("expected limit=200 on first page, got %q", got)
			}
			body := `{"data":[{"type":"appStoreVersionLocalizations","id":"loc-1"}],` +
				`"included":[{"type":"appScreenshotSets","id":"set-1"}],"links":{"next":"` + secondURL + `"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 2:
			if req.URL.String() != secondURL {
				t.Fatalf("unexpected second request: %s", req.URL.String())
			}
			body := `{"data":[{"type":"appStoreVersionLocalizations","id":"loc-2"}],` +
				`"included":[{"type":"appScreenshotSets","id":"set-2"}],"links":{"next":""}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected extra request: %s", req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"localizations", "list",
			"--version", "version-1",
			"--include", "appScreenshotSets",
			"--paginate",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if requestCount != 2 {
		t.Fatalf("expected exactly 2 requests, got %d", requestCount)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{`"id":"loc-1"`, `"id":"loc-2"`, `"id":"set-1"`, `"id":"set-2"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %s in aggregated output, got %q", want, stdout)
		}
	}
}

func TestLocalizationsListIncludeValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "unsupported include value",
			args:    []string{"localizations", "list", "--version", "version-1", "--include", "appScreenshotSets,builds"},
			wantErr: "--include must be one of: appStoreVersion, appScreenshotSets, appPreviewSets, searchKeywords",
		},
		{
			name:    "empty include value",
			args:    []string{"localizations", "list", "--version", "version-1", "--include", ""},
			wantErr: "--include must not be empty",
		},
		{
			name:    "repeated include flag",
			args:    []string{"localizations", "list", "--version", "version-1", "--include", "appScreenshotSets", "--include", "appPreviewSets"},
			wantErr: `--include specified multiple times; pass one comma-separated list, for example --include "appScreenshotSets,appPreviewSets"`,
		},
		{
			name: "include with next",
			args: []string{
				"localizations", "list",
				"--version", "version-1",
				"--include", "appScreenshotSets",
				"--next", "https://api.appstoreconnect.apple.com/v1/appStoreVersions/version-1/appStoreVersionLocalizations?cursor=AQ",
			},
			wantErr: "--next cannot be combined with --include",
		},
		{
			name:    "include with app-info type",
			args:    []string{"localizations", "list", "--app", "app-1", "--type", "app-info", "--include", "appStoreVersion"},
			wantErr: "--include requires --type version",
		},
		{
			name:    "app-info relationship with app-info type",
			args:    []string{"localizations", "list", "--app", "app-1", "--type", "app-info", "--include", "appInfo"},
			wantErr: "--include requires --type version",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			originalTransport := http.DefaultTransport
			t.Cleanup(func() {
				http.DefaultTransport = originalTransport
			})
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("unexpected request for invalid input: %s", req.URL.String())
				return nil, nil
			})

			_, stderr := captureOutput(t, func() {
				if code := cmd.Run(test.args, "1.0.0"); code != cmd.ExitUsage {
					t.Fatalf("expected exit code %d, got %d", cmd.ExitUsage, code)
				}
			})
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected stderr to contain %q, got %q", test.wantErr, stderr)
			}
		})
	}
}
