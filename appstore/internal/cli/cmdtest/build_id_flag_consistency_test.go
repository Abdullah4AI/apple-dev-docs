package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

const legacyBuildFlagWarning = "Warning: `--build` is deprecated. Use `--build-id`."

// buildIDCanonicalCommands lists every command whose build selector was spelled
// `--build` while the rest of the CLI already documented `--build-id`.
var buildIDCanonicalCommands = [][]string{
	{"review", "submit"},
	{"publish", "testflight"},
	{"validate", "testflight"},
	{"release", "stage"},
	{"build-localizations", "list"},
	{"build-localizations", "create"},
	{"build-bundles", "list"},
	{"testflight", "crashes", "list"},
	{"testflight", "feedback", "list"},
	{"performance", "metrics", "view"},
	{"performance", "diagnostics", "list"},
	{"performance", "download"},
	{"encryption", "declarations", "list"},
	{"encryption", "declarations", "assign-builds"},
	{"apps", "app-encryption-declarations", "list"},
}

func TestBuildIDIsTheCanonicalSpellingAcrossBuildSelectors(t *testing.T) {
	root := RootCommand("1.2.3")

	for _, path := range buildIDCanonicalCommands {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			command := findSubcommand(root, path...)
			if command == nil {
				t.Fatalf("command %q not found", strings.Join(path, " "))
			}
			if command.FlagSet.Lookup("build-id") == nil {
				t.Fatalf("canonical flag --build-id not registered on %q", strings.Join(path, " "))
			}
			if command.FlagSet.Lookup("build") == nil {
				t.Fatalf("compatibility flag --build must keep working on %q", strings.Join(path, " "))
			}

			usage := command.UsageFunc(command)
			if !strings.Contains(usage, "\n  --build-id ") {
				t.Fatalf("expected --build-id documented in help for %q: %q", strings.Join(path, " "), usage)
			}
			if strings.Contains(usage, "\n  --build ") {
				t.Fatalf("compatibility alias --build should stay hidden from help for %q: %q", strings.Join(path, " "), usage)
			}
		})
	}
}

func runBuildIDFlagCommand(t *testing.T, args []string) (string, string, error) {
	t.Helper()

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse(args); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	return stdout, stderr, runErr
}

func installReviewSubmitAlreadySubmittedTransport(t *testing.T, requestCount *int) {
	t.Helper()

	installDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		*requestCount++
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/version-1":
			return jsonResponse(http.StatusOK, `{
				"data":{
					"type":"appStoreVersions",
					"id":"version-1",
					"attributes":{"platform":"IOS","versionString":"1.2.3"},
					"relationships":{"app":{"data":{"type":"apps","id":"app-1"}}}
				},
				"included":[{"type":"apps","id":"app-1","attributes":{"bundleId":"app-1","name":"App One"}}]
			}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/version-1/appStoreVersionSubmission":
			return jsonResponse(http.StatusOK, `{"data":{"type":"appStoreVersionSubmissions","id":"submission-1"}}`)
		default:
			t.Fatalf("unexpected request: %s %s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
			return nil, nil
		}
	}))
}

func TestReviewSubmitBuildIDAliasMatchesCanonical(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	requestCount := 0
	installReviewSubmitAlreadySubmittedTransport(t, &requestCount)

	canonicalStdout, canonicalStderr, canonicalErr := runBuildIDFlagCommand(t, []string{
		"review", "submit",
		"--app", "app-1",
		"--version-id", "version-1",
		"--build-id", "build-1",
		"--dry-run",
		"--output", "json",
	})
	if canonicalErr != nil {
		t.Fatalf("canonical run error: %v", canonicalErr)
	}
	if !strings.Contains(canonicalStdout, `"buildId":"build-1"`) {
		t.Fatalf("expected --build-id to populate the build selector, got %q", canonicalStdout)
	}
	if strings.Contains(canonicalStderr, legacyBuildFlagWarning) {
		t.Fatalf("did not expect a deprecation warning for the canonical spelling, got %q", canonicalStderr)
	}

	aliasStdout, aliasStderr, aliasErr := runBuildIDFlagCommand(t, []string{
		"review", "submit",
		"--app", "app-1",
		"--version-id", "version-1",
		"--build", "build-1",
		"--dry-run",
		"--output", "json",
	})
	if aliasErr != nil {
		t.Fatalf("alias run error: %v", aliasErr)
	}
	if aliasStdout != canonicalStdout {
		t.Fatalf("expected identical stdout, canonical=%q alias=%q", canonicalStdout, aliasStdout)
	}
	if !strings.Contains(aliasStderr, legacyBuildFlagWarning) {
		t.Fatalf("expected legacy build warning, got %q", aliasStderr)
	}
	if requestCount != 4 {
		t.Fatalf("expected four requests, got %d", requestCount)
	}
}

func TestBuildIDAliasConflictErrors(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "review submit",
			args: []string{"review", "submit", "--app", "app-1", "--version-id", "version-1", "--build-id", "build-canonical", "--build", "build-legacy", "--dry-run"},
		},
		{
			name: "validate testflight",
			args: []string{"validate", "testflight", "--app", "app-1", "--build-id", "build-canonical", "--build", "build-legacy"},
		},
		{
			name: "build-bundles list",
			args: []string{"build-bundles", "list", "--build-id", "build-canonical", "--build", "build-legacy"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, runErr := runBuildIDFlagCommand(t, test.args)
			if !errors.Is(runErr, flag.ErrHelp) {
				t.Fatalf("expected ErrHelp, got %v", runErr)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, "Error: --build conflicts with --build-id; use only --build-id") {
				t.Fatalf("expected conflicting build selector error, got %q", stderr)
			}
		})
	}
}

func TestBuildIDRequiredErrorsNameCanonicalFlag(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	tests := []struct {
		name string
		args []string
	}{
		{name: "review submit", args: []string{"review", "submit", "--app", "app-1", "--version", "1.2.3", "--confirm"}},
		{name: "validate testflight", args: []string{"validate", "testflight", "--app", "app-1"}},
		{name: "build-bundles list", args: []string{"build-bundles", "list"}},
		{name: "build-localizations create", args: []string{"build-localizations", "create", "--locale", "en-US"}},
		{name: "performance diagnostics list", args: []string{"performance", "diagnostics", "list"}},
		{name: "performance metrics view", args: []string{"performance", "metrics", "view"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, runErr := runBuildIDFlagCommand(t, test.args)
			if !errors.Is(runErr, flag.ErrHelp) {
				t.Fatalf("expected ErrHelp, got %v", runErr)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, "Error: --build-id is required") {
				t.Fatalf("expected --build-id required error, got %q", stderr)
			}
		})
	}
}
