package cmdtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

const releaseTypeValues = "MANUAL, AFTER_APPROVAL, SCHEDULED"

func clearASCAuth(t *testing.T) {
	t.Helper()
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_PROFILE", "")
	t.Setenv("ASC_KEY_ID", "")
	t.Setenv("ASC_ISSUER_ID", "")
	t.Setenv("ASC_PRIVATE_KEY_PATH", "")
	t.Setenv("ASC_PRIVATE_KEY", "")
	t.Setenv("ASC_PRIVATE_KEY_B64", "")
	t.Setenv("ASC_STRICT_AUTH", "")
}

func TestVersionsViewSendsExactSupportedIncludeQuery(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	const include = "app,appStoreVersionLocalizations,build,appStoreVersionPhasedRelease"
	stubTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/appStoreVersions/version-1" {
			t.Fatalf("request = %s %s, want GET /v1/appStoreVersions/version-1", req.Method, req.URL.Path)
		}
		if got := req.URL.Query().Get("include"); got != include {
			t.Fatalf("include query = %q, want %q", got, include)
		}
		return jsonResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"version-1","attributes":{"versionString":"1.0","platform":"IOS"}},"included":[]}`)
	})

	root := RootCommand("test")
	root.FlagSet.SetOutput(io.Discard)
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"versions", "view",
			"--version-id", "version-1",
			"--include", include,
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("unmarshal stdout: %v; stdout=%q", err, stdout)
	}
	if payload.Data.ID != "version-1" {
		t.Fatalf("version id = %q, want version-1", payload.Data.ID)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestVersionsReleaseTypeValidationBeforeAuth(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		prefix string
	}{
		{
			name:   "create",
			args:   []string{"versions", "create", "--app", "app-1", "--version", "1.0", "--release-type", "INVALID"},
			prefix: "versions create",
		},
		{
			name:   "update",
			args:   []string{"versions", "update", "--version-id", "version-1", "--release-type", "INVALID"},
			prefix: "versions update",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearASCAuth(t)
			stdout, stderr := captureOutput(t, func() {
				code := rootcmd.Run(test.args, "test")
				if code != rootcmd.ExitUsage {
					t.Fatalf("exit code = %d, want %d", code, rootcmd.ExitUsage)
				}
			})

			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			want := "Error: " + test.prefix + ": --release-type must be one of: " + releaseTypeValues
			if !strings.Contains(stderr, want) {
				t.Fatalf("stderr = %q, want %q", stderr, want)
			}
			if strings.Contains(stderr, "missing authentication") {
				t.Fatalf("stderr = %q, validation must run before auth", stderr)
			}
		})
	}
}

func TestVersionsReleaseTypePayloads(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		method     string
		path       string
		responseID string
	}{
		{
			name:       "create",
			args:       []string{"versions", "create", "--app", "app-1", "--version", "2.0", "--release-type", "scheduled", "--output", "json"},
			method:     http.MethodPost,
			path:       "/v1/appStoreVersions",
			responseID: "version-new",
		},
		{
			name:       "update",
			args:       []string{"versions", "update", "--version-id", "version-1", "--release-type", "scheduled", "--output", "json"},
			method:     http.MethodPatch,
			path:       "/v1/appStoreVersions/version-1",
			responseID: "version-1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
			requestErr := make(chan error, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != test.method || req.URL.Path != test.path {
					requestErr <- fmt.Errorf("request = %s %s, want %s %s", req.Method, req.URL.Path, test.method, test.path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}
				var request struct {
					Data struct {
						Attributes map[string]any `json:"attributes"`
					} `json:"data"`
				}
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
					requestErr <- fmt.Errorf("decode request: %w", err)
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
				if got := request.Data.Attributes["releaseType"]; got != "SCHEDULED" {
					requestErr <- fmt.Errorf("releaseType = %#v, want SCHEDULED", got)
					http.Error(w, "invalid release type", http.StatusBadRequest)
					return
				}
				requestErr <- nil
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"data":{"type":"appStoreVersions","id":"`+test.responseID+`","attributes":{"versionString":"2.0","platform":"IOS"}}}`)
			}))
			t.Cleanup(server.Close)

			serverURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("parse server URL: %v", err)
			}
			installDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
				cloned := req.Clone(req.Context())
				cloned.URL.Scheme = serverURL.Scheme
				cloned.URL.Host = serverURL.Host
				return server.Client().Transport.RoundTrip(cloned)
			}))

			root := RootCommand("test")
			root.FlagSet.SetOutput(io.Discard)
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				if err := root.Run(context.Background()); err != nil {
					t.Fatalf("run error: %v", err)
				}
			})

			var result struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(stdout), &result); err != nil {
				t.Fatalf("unmarshal stdout: %v; stdout=%q", err, stdout)
			}
			if result.ID != test.responseID {
				t.Fatalf("response id = %q, want %q", result.ID, test.responseID)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if err := <-requestErr; err != nil {
				t.Fatal(err)
			}
		})
	}
}
