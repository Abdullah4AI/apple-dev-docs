package cmdtest

import (
	"context"
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

// buildsListQuerySurfaceRequest records what the CLI sent, so tests can assert
// both the emitted query and that rejected invocations sent nothing at all.
type buildsListQuerySurfaceRequest struct {
	calls int
	path  string
	query url.Values
}

// buildsListQuerySurfaceStub installs a transport that captures the builds
// request the CLI issues and replies with a minimal envelope.
func buildsListQuerySurfaceStub(t *testing.T) *buildsListQuerySurfaceRequest {
	t.Helper()

	setupAuth(t)
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	captured := &buildsListQuerySurfaceRequest{}

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		captured.calls++
		captured.path = req.URL.Path
		captured.query = req.URL.Query()
		body := `{"data":[{"type":"builds","id":"build-1","attributes":{"version":"42"}}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	return captured
}

// assertNoRequest fails when a rejected invocation reached the network, proving
// validation runs before any side effect.
func (r *buildsListQuerySurfaceRequest) assertNoRequest(t *testing.T) {
	t.Helper()

	if r.calls != 0 {
		t.Fatalf("expected validation to short-circuit before any request, got %d call(s) to %s?%s", r.calls, r.path, r.query.Encode())
	}
}

func runBuildsListQuerySurface(t *testing.T, args ...string) (string, string, error) {
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

func TestBuildsListBetaReviewStateEmitsFilter(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	stdout, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "list",
		"--app", "123456789",
		"--beta-review-state", "WAITING_FOR_REVIEW,IN_REVIEW",
	)
	if err != nil {
		t.Fatalf("run error: %v (stderr=%q)", err, stderr)
	}

	if captured.path != "/v1/builds" {
		t.Fatalf("expected /v1/builds path, got %q", captured.path)
	}
	if got := captured.query.Get("filter[betaAppReviewSubmission.betaReviewState]"); got != "WAITING_FOR_REVIEW,IN_REVIEW" {
		t.Fatalf("expected beta review state filter, got %q", got)
	}
	if got := captured.query.Get("filter[app]"); got != "123456789" {
		t.Fatalf("expected filter[app]=123456789, got %q", got)
	}
	if !strings.Contains(stdout, `"id":"build-1"`) {
		t.Fatalf("expected build envelope, got %q", stdout)
	}
}

func TestBuildsListBetaReviewStateNormalizesCase(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "list",
		"--app", "123456789",
		"--beta-review-state", "approved",
	)
	if err != nil {
		t.Fatalf("run error: %v (stderr=%q)", err, stderr)
	}

	if got := captured.query.Get("filter[betaAppReviewSubmission.betaReviewState]"); got != "APPROVED" {
		t.Fatalf("expected normalized APPROVED, got %q", got)
	}
}

func TestBuildsListBetaReviewStateRejectsUnknownValue(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "list",
		"--app", "123456789",
		"--beta-review-state", "WAITING_FOR_REVIEW,PENDING",
	)

	if got := rootcmd.ExitCodeFromError(err); got != rootcmd.ExitUsage {
		t.Fatalf("exit code = %d, want %d (err=%v)", got, rootcmd.ExitUsage, err)
	}
	for _, want := range []string{"--beta-review-state", "WAITING_FOR_REVIEW", "IN_REVIEW", "REJECTED", "APPROVED"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected stderr to mention %q, got %q", want, stderr)
		}
	}
	captured.assertNoRequest(t)
}

func TestBuildsBetaReviewStateRejectsBlankCSV(t *testing.T) {
	for _, command := range []string{"list", "count"} {
		t.Run(command, func(t *testing.T) {
			captured := buildsListQuerySurfaceStub(t)

			_, stderr, err := runBuildsListQuerySurface(
				t,
				"builds", command,
				"--app", "123456789",
				"--beta-review-state", ",",
			)

			if got := rootcmd.ExitCodeFromError(err); got != rootcmd.ExitUsage {
				t.Fatalf("exit code = %d, want %d (err=%v)", got, rootcmd.ExitUsage, err)
			}
			if !strings.Contains(stderr, "--beta-review-state must be a comma-separated list of") {
				t.Fatalf("expected beta review state validation error, got %q", stderr)
			}
			captured.assertNoRequest(t)
		})
	}
}

func TestBuildsListIncludeDefaultsToPreReleaseVersion(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(t, "builds", "list", "--app", "123456789")
	if err != nil {
		t.Fatalf("run error: %v (stderr=%q)", err, stderr)
	}

	if got := captured.query.Get("include"); got != "preReleaseVersion" {
		t.Fatalf("expected default include=preReleaseVersion, got %q", got)
	}
}

func TestBuildsListIncludeUnionsWithPreReleaseVersion(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "list",
		"--app", "123456789",
		"--include", "app,buildBetaDetail",
	)
	if err != nil {
		t.Fatalf("run error: %v (stderr=%q)", err, stderr)
	}

	if got := captured.query.Get("include"); got != "preReleaseVersion,app,buildBetaDetail" {
		t.Fatalf("expected include to union the table default, got %q", got)
	}
}

func TestBuildsListIncludeDoesNotDuplicatePreReleaseVersion(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "list",
		"--app", "123456789",
		"--include", "preReleaseVersion,betaGroups,preReleaseVersion",
	)
	if err != nil {
		t.Fatalf("run error: %v (stderr=%q)", err, stderr)
	}

	if got := captured.query.Get("include"); got != "preReleaseVersion,betaGroups" {
		t.Fatalf("expected deduplicated include, got %q", got)
	}
}

func TestBuildsListIncludeRejectsUnknownValue(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "list",
		"--app", "123456789",
		"--include", "app,buildBundle",
	)

	if got := rootcmd.ExitCodeFromError(err); got != rootcmd.ExitUsage {
		t.Fatalf("exit code = %d, want %d (err=%v)", got, rootcmd.ExitUsage, err)
	}
	for _, want := range []string{
		"--include",
		"preReleaseVersion",
		"individualTesters",
		"betaGroups",
		"betaBuildLocalizations",
		"appEncryptionDeclaration",
		"betaAppReviewSubmission",
		"app",
		"buildBetaDetail",
		"appStoreVersion",
		"icons",
		"buildBundles",
		"buildUpload",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected stderr to mention %q, got %q", want, stderr)
		}
	}
	captured.assertNoRequest(t)
}

func TestBuildsListIncludeRejectsBlankCSV(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "list",
		"--app", "123456789",
		"--include", ",",
	)

	if got := rootcmd.ExitCodeFromError(err); got != rootcmd.ExitUsage {
		t.Fatalf("exit code = %d, want %d (err=%v)", got, rootcmd.ExitUsage, err)
	}
	if !strings.Contains(stderr, "--include must be a comma-separated list of") {
		t.Fatalf("expected include validation error, got %q", stderr)
	}
	captured.assertNoRequest(t)
}

func TestBuildsListRejectsQueryFlagsCombinedWithNext(t *testing.T) {
	const nextURL = "https://api.appstoreconnect.apple.com/v1/builds?cursor=PAGE2&filter%5Bapp%5D=123456789&include=preReleaseVersion"

	for _, testCase := range []struct {
		name string
		flag string
		args []string
	}{
		{name: "beta review state", flag: "--beta-review-state", args: []string{"--beta-review-state", "APPROVED"}},
		{name: "include", flag: "--include", args: []string{"--include", "betaGroups"}},
		{name: "sort", flag: "--sort", args: []string{"--sort", "version"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			captured := buildsListQuerySurfaceStub(t)

			args := append([]string{"builds", "list", "--next", nextURL}, testCase.args...)
			_, stderr, err := runBuildsListQuerySurface(t, args...)

			if got := rootcmd.ExitCodeFromError(err); got != rootcmd.ExitUsage {
				t.Fatalf("exit code = %d, want %d (err=%v)", got, rootcmd.ExitUsage, err)
			}
			if !strings.Contains(stderr, "--next cannot be combined with "+testCase.flag) {
				t.Fatalf("expected stderr to reject %s with --next, got %q", testCase.flag, stderr)
			}
			captured.assertNoRequest(t)
		})
	}
}

func TestBuildsListNextAloneStillPaginates(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "list",
		"--next", "https://api.appstoreconnect.apple.com/v1/builds?cursor=PAGE2&filter%5Bapp%5D=123456789&include=preReleaseVersion",
	)
	if err != nil {
		t.Fatalf("run error: %v (stderr=%q)", err, stderr)
	}

	if captured.path != "/v1/builds" {
		t.Fatalf("expected /v1/builds path, got %q", captured.path)
	}
	// The whole next URL must survive, not just the cursor: dropping filter[app]
	// or include here would silently widen the page or break the table renderer,
	// and adding anything would mean the URL was rebuilt rather than followed.
	wantQuery := map[string]string{
		"cursor":      "PAGE2",
		"filter[app]": "123456789",
		"include":     "preReleaseVersion",
	}
	for param, want := range wantQuery {
		if got := captured.query.Get(param); got != want {
			t.Fatalf("expected %s=%q on the followed next URL, got %q (full query %s)", param, want, got, captured.query.Encode())
		}
	}
	if len(captured.query) != len(wantQuery) {
		t.Fatalf("expected exactly the next URL's %d parameters, got %s", len(wantQuery), captured.query.Encode())
	}
}

func TestBuildsListSortAcceptsEveryDocumentedKey(t *testing.T) {
	for _, sortValue := range []string{
		"version",
		"-version",
		"uploadedDate",
		"-uploadedDate",
		"preReleaseVersion",
		"-preReleaseVersion",
	} {
		t.Run(sortValue, func(t *testing.T) {
			captured := buildsListQuerySurfaceStub(t)

			_, stderr, err := runBuildsListQuerySurface(
				t,
				"builds", "list",
				"--app", "123456789",
				"--sort", sortValue,
			)
			if err != nil {
				t.Fatalf("run error: %v (stderr=%q)", err, stderr)
			}

			if got := captured.query.Get("sort"); got != sortValue {
				t.Fatalf("expected sort=%q, got %q", sortValue, got)
			}
		})
	}
}

// builds count documents filter parity with builds list, so the beta review
// state filter has to reach the count request too.
func TestBuildsCountBetaReviewStateEmitsFilter(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "count",
		"--app", "123456789",
		"--beta-review-state", "rejected",
	)
	if err != nil {
		t.Fatalf("run error: %v (stderr=%q)", err, stderr)
	}

	if captured.path != "/v1/builds" {
		t.Fatalf("expected /v1/builds path, got %q", captured.path)
	}
	if got := captured.query.Get("filter[betaAppReviewSubmission.betaReviewState]"); got != "REJECTED" {
		t.Fatalf("expected normalized REJECTED filter, got %q", got)
	}
}

func TestBuildsCountBetaReviewStateRejectsUnknownValue(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "count",
		"--app", "123456789",
		"--beta-review-state", "PENDING",
	)

	if got := rootcmd.ExitCodeFromError(err); got != rootcmd.ExitUsage {
		t.Fatalf("exit code = %d, want %d (err=%v)", got, rootcmd.ExitUsage, err)
	}
	wantError := "--beta-review-state must be a comma-separated list of: WAITING_FOR_REVIEW, IN_REVIEW, REJECTED, APPROVED"
	if err == nil || !errors.Is(err, flag.ErrHelp) || err.Error() != wantError {
		t.Fatalf("error = %v, want usage error %q (stderr=%q)", err, wantError, stderr)
	}
	if !strings.Contains(stderr, wantError) {
		t.Fatalf("expected stderr to contain %q, got %q", wantError, stderr)
	}
	captured.assertNoRequest(t)
}

func TestBuildsListSortRejectsUnknownKey(t *testing.T) {
	captured := buildsListQuerySurfaceStub(t)

	_, stderr, err := runBuildsListQuerySurface(
		t,
		"builds", "list",
		"--app", "123456789",
		"--sort", "expirationDate",
	)

	wantError := "builds: --sort must be one of: version, -version, uploadedDate, -uploadedDate, preReleaseVersion, -preReleaseVersion"
	if got := rootcmd.ExitCodeFromError(err); got != rootcmd.ExitUsage {
		t.Fatalf("exit code = %d, want %d (err=%v)", got, rootcmd.ExitUsage, err)
	}
	if err == nil || !errors.Is(err, flag.ErrHelp) || err.Error() != wantError {
		t.Fatalf("error = %v, want usage error %q (stderr=%q)", err, wantError, stderr)
	}
	if !strings.Contains(stderr, wantError) {
		t.Fatalf("expected stderr to contain %q, got %q", wantError, stderr)
	}
	captured.assertNoRequest(t)
}
