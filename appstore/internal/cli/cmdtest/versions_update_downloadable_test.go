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
)

// The public write path that replaces the retired read's missing write.
// `downloadable` is nullable on AppStoreVersionUpdateRequest, so the flag is
// tri-state: unset sends nothing, true and false send the boolean.
func TestVersionsUpdateSendsDownloadable(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "true", args: []string{"--downloadable", "true"}, want: true},
		{name: "false", args: []string{"--downloadable", "false", "--confirm"}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			var body map[string]any
			stubTransport(t, func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch || req.URL.Path != "/v1/appStoreVersions/version-1" {
					t.Fatalf("request = %s %s, want PATCH /v1/appStoreVersions/version-1", req.Method, req.URL.Path)
				}
				payload, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("read request body: %v", err)
				}
				if err := json.Unmarshal(payload, &body); err != nil {
					t.Fatalf("unmarshal request body: %v; body=%s", err, payload)
				}
				return jsonResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"version-1","attributes":{"versionString":"1.2.3","platform":"IOS","appVersionState":"READY_FOR_DISTRIBUTION","downloadable":`+boolLiteral(test.want)+`}}}`)
			})

			args := append([]string{"versions", "update", "--version-id", "version-1", "--output", "json"}, test.args...)
			root := RootCommand("test")
			root.FlagSet.SetOutput(io.Discard)
			stdout, _ := captureOutput(t, func() {
				if err := root.Parse(args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				if err := root.Run(context.Background()); err != nil {
					t.Fatalf("run error: %v", err)
				}
			})

			data, _ := body["data"].(map[string]any)
			attributes, _ := data["attributes"].(map[string]any)
			got, ok := attributes["downloadable"]
			if !ok {
				t.Fatalf("request attributes missing downloadable: %v", attributes)
			}
			if got != any(test.want) {
				t.Fatalf("downloadable = %v, want %v", got, test.want)
			}

			var result map[string]any
			if err := json.Unmarshal([]byte(stdout), &result); err != nil {
				t.Fatalf("unmarshal stdout: %v; stdout=%q", err, stdout)
			}
			if result["downloadable"] != any(test.want) {
				t.Fatalf("receipt downloadable = %v, want %v; stdout=%q", result["downloadable"], test.want, stdout)
			}
		})
	}
}

// An unset --downloadable must not send the attribute at all, so an unrelated
// update cannot flip download availability by accident.
func TestVersionsUpdateOmitsDownloadableWhenUnset(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	var body map[string]any
	stubTransport(t, func(req *http.Request) (*http.Response, error) {
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(payload, &body); err != nil {
			t.Fatalf("unmarshal request body: %v; body=%s", err, payload)
		}
		return jsonResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"version-1","attributes":{"versionString":"1.2.3","platform":"IOS"}}}`)
	})

	root := RootCommand("test")
	root.FlagSet.SetOutput(io.Discard)
	captureOutput(t, func() {
		if err := root.Parse([]string{"versions", "update", "--version-id", "version-1", "--copyright", "2026 Example", "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	data, _ := body["data"].(map[string]any)
	attributes, _ := data["attributes"].(map[string]any)
	if _, ok := attributes["downloadable"]; ok {
		t.Fatalf("unset --downloadable still sent the attribute: %v", attributes)
	}
}

// Making a released version undownloadable is not reversible from every state,
// so --downloadable false is a destructive write that requires --confirm.
func TestVersionsUpdateDownloadableFalseRequiresConfirm(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	stubTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("must fail before HTTP: %s %s", req.Method, req.URL.String())
		return nil, errors.New("unexpected request")
	}))

	stdout, stderr, runErr := runCommand(t, []string{"versions", "update", "--version-id", "version-1", "--downloadable", "false"})
	if runErr == nil {
		t.Fatal("run error = nil, want usage error")
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "--confirm is required") {
		t.Fatalf("stderr = %q, want --confirm guidance", stderr)
	}
}

// --confirm only means anything alongside --downloadable false. Accepting it
// anywhere else would silently ignore a flag the caller passed deliberately.
func TestVersionsUpdateRejectsConfirmWithoutDownloadableFalse(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "downloadable true", args: []string{"--downloadable", "true", "--confirm"}},
		{name: "unrelated update", args: []string{"--copyright", "2026 Example", "--confirm"}},
		// An explicit --confirm=false is still a supplied flag with nothing to
		// confirm, so it is rejected rather than quietly dropped.
		{name: "explicit false", args: []string{"--copyright", "2026 Example", "--confirm=false"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			stubTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("must fail before HTTP: %s %s", req.Method, req.URL.String())
				return nil, errors.New("unexpected request")
			}))

			args := append([]string{"versions", "update", "--version-id", "version-1"}, test.args...)
			stdout, stderr, runErr := runCommand(t, args)
			if runErr == nil {
				t.Fatal("run error = nil, want usage error")
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, "--confirm applies only to --downloadable false") {
				t.Fatalf("stderr = %q, want the --confirm misuse error", stderr)
			}
		})
	}
}

// --confirm=false alongside --downloadable false is not confirmation, so the
// destructive write is still refused.
func TestVersionsUpdateDownloadableFalseRejectsExplicitConfirmFalse(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	stubTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("must fail before HTTP: %s %s", req.Method, req.URL.String())
		return nil, errors.New("unexpected request")
	}))

	stdout, stderr, runErr := runCommand(t, []string{"versions", "update", "--version-id", "version-1", "--downloadable", "false", "--confirm=false"})
	if runErr == nil {
		t.Fatal("run error = nil, want usage error")
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "--confirm is required") {
		t.Fatalf("stderr = %q, want --confirm guidance", stderr)
	}
}

// Boolean flags do not consume a following spaced value. Reject the leftover
// operand before a bare --confirm can authorize the destructive write.
func TestVersionsUpdateRejectsSpacedConfirmFalseBeforeHTTP(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	stubTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("must fail before HTTP: %s %s", req.Method, req.URL.String())
		return nil, errors.New("unexpected request")
	}))

	stdout, stderr, runErr := runCommand(t, []string{"versions", "update", "--version-id", "version-1", "--downloadable", "false", "--confirm", "false"})
	if runErr == nil {
		t.Fatal("run error = nil, want usage error")
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "unexpected argument(s): false") {
		t.Fatalf("stderr = %q, want positional-argument guidance", stderr)
	}
}

// --downloadable alone satisfies the "at least one field" guard; it must not be
// reported as a no-op update.
func TestVersionsUpdateAcceptsDownloadableAsTheOnlyField(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	sent := false
	stubTransport(t, func(req *http.Request) (*http.Response, error) {
		sent = true
		return jsonResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"version-1","attributes":{"versionString":"1.2.3","platform":"IOS","downloadable":true}}}`)
	})

	root := RootCommand("test")
	root.FlagSet.SetOutput(io.Discard)
	captureOutput(t, func() {
		if err := root.Parse([]string{"versions", "update", "--version-id", "version-1", "--downloadable", "true", "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if !sent {
		t.Fatal("expected a PATCH request for --downloadable alone")
	}
}

func boolLiteral(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
