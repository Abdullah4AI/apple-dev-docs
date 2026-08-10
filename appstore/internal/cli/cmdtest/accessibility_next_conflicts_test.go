package cmdtest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestAccessibilityListRejectsNextQueryFlagsBeforeAuth(t *testing.T) {
	t.Cleanup(func() { shared.SetSelectedProfile("") })
	t.Setenv("ASC_APP_ID", "")

	const nextURL = "https://api.appstoreconnect.apple.com/v1/apps/app-1/accessibilityDeclarations?cursor=next"
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "app before next",
			args:    []string{"accessibility", "list", "--app", "app-2", "--next", nextURL},
			wantErr: "accessibility list: --next cannot be combined with --app",
		},
		{
			name:    "explicit empty app after next",
			args:    []string{"accessibility", "list", "--next", nextURL, "--app", ""},
			wantErr: "accessibility list: --next cannot be combined with --app",
		},
		{
			name:    "device family after next",
			args:    []string{"accessibility", "list", "--next", nextURL, "--device-family", "IPHONE"},
			wantErr: "accessibility list: --next cannot be combined with --device-family",
		},
		{
			name:    "state before next",
			args:    []string{"accessibility", "list", "--state", "PUBLISHED", "--next", nextURL},
			wantErr: "accessibility list: --next cannot be combined with --state",
		},
		{
			name:    "explicit empty fields after next",
			args:    []string{"accessibility", "list", "--next", nextURL, "--fields", ""},
			wantErr: "accessibility list: --next cannot be combined with --fields",
		},
		{
			name:    "explicit zero limit before next",
			args:    []string{"accessibility", "list", "--limit", "0", "--next", nextURL},
			wantErr: "accessibility list: --next cannot be combined with --limit",
		},
		{
			name:    "out of range limit after next",
			args:    []string{"accessibility", "list", "--next", nextURL, "--limit", "201"},
			wantErr: "accessibility list: --next cannot be combined with --limit",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientFactoryCalled := false
			restore := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
				clientFactoryCalled = true
				return nil, errors.New("client factory must not run during validation")
			})
			defer restore()

			assertUsageExit(t, test.args, test.wantErr)
			if clientFactoryCalled {
				t.Fatal("client factory ran before --next conflict validation")
			}
		})
	}
}

func TestAccessibilityListInvalidNextPrecedesLimitConflict(t *testing.T) {
	stdout, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{
			"accessibility", "list",
			"--next", "http://api.appstoreconnect.apple.com/v1/apps/app-1/accessibilityDeclarations?cursor=next",
			"--limit", "201",
		}, "1.2.3")
		if code != rootcmd.ExitError {
			t.Fatalf("exit code = %d, want %d", code, rootcmd.ExitError)
		}
	})
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "accessibility list: --next must be an App Store Connect URL") {
		t.Fatalf("stderr = %q, want invalid --next error", stderr)
	}
	if strings.Contains(stderr, "--limit") {
		t.Fatalf("stderr = %q, want --next validation to take precedence", stderr)
	}
}

func TestAccessibilityListNextOnlyUsesCursorURL(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_APP_ID", "env-app")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	const nextURL = "https://api.appstoreconnect.apple.com/v1/apps/app-1/accessibilityDeclarations?cursor=page-2&filter%5Bstate%5D=PUBLISHED&limit=5"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/apps/app-1/accessibilityDeclarations" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		query := req.URL.Query()
		if got := query.Get("cursor"); got != "page-2" {
			t.Errorf("cursor = %q, want page-2", got)
		}
		if got := query.Get("filter[state]"); got != "PUBLISHED" {
			t.Errorf("filter[state] = %q, want PUBLISHED", got)
		}
		if got := query.Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"type":"accessibilityDeclarations","id":"decl-next"}],"links":{"next":""}}`)
	}))
	t.Cleanup(server.Close)
	installDefaultTransportForServer(t, server)

	root := RootCommand("1.2.3")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"accessibility", "list", "--next", nextURL, "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, `"id":"decl-next"`) {
		t.Fatalf("stdout = %q, want cursor response", stdout)
	}
}

func TestAccessibilityListFilterOnlyBuildsDocumentedQuery(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/apps/app-1/accessibilityDeclarations" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		want := url.Values{
			"fields[accessibilityDeclarations]": {"deviceFamily,state"},
			"filter[deviceFamily]":              {"IPHONE,IPAD"},
			"filter[state]":                     {"DRAFT"},
			"limit":                             {"25"},
		}
		if got := req.URL.Query(); got.Encode() != want.Encode() {
			t.Errorf("query = %q, want %q", got.Encode(), want.Encode())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"type":"accessibilityDeclarations","id":"decl-filtered"}],"links":{"next":""}}`)
	}))
	t.Cleanup(server.Close)
	installDefaultTransportForServer(t, server)

	root := RootCommand("1.2.3")
	stdout, stderr := captureOutput(t, func() {
		args := []string{
			"accessibility", "list",
			"--app", "app-1",
			"--device-family", "iphone,ipad",
			"--state", "draft",
			"--fields", "deviceFamily,state",
			"--limit", "25",
			"--output", "json",
		}
		if err := root.Parse(args); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, `"id":"decl-filtered"`) {
		t.Fatalf("stdout = %q, want filtered response", stdout)
	}
}
