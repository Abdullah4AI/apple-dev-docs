package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/validate"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/validation"
)

type validateSubscriptionsFixture struct {
	groups               string
	subscriptionsByGroup map[string]string
}

func newValidateSubscriptionsClient(t *testing.T, fixture validateSubscriptionsFixture) *asc.Client {
	t.Helper()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.p8")
	writeECDSAPEM(t, keyPath)

	notFound := `{"errors":[{"code":"NOT_FOUND","title":"Not Found","detail":"resource not found"}]}`

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return jsonResponse(http.StatusMethodNotAllowed, `{"errors":[{"status":405}]}`)
		}

		path := req.URL.Path
		switch {
		case path == "/v1/apps/app-1/subscriptionGroups":
			return jsonResponse(http.StatusOK, fixture.groups)
		case strings.HasPrefix(path, "/v1/subscriptionGroups/") && strings.HasSuffix(path, "/subscriptions"):
			groupID := strings.TrimSuffix(strings.TrimPrefix(path, "/v1/subscriptionGroups/"), "/subscriptions")
			if body, ok := fixture.subscriptionsByGroup[groupID]; ok {
				return jsonResponse(http.StatusOK, body)
			}
			return jsonResponse(http.StatusOK, `{"data":[]}`)
		default:
			return jsonResponse(http.StatusNotFound, notFound)
		}
	})

	httpClient := &http.Client{Transport: transport}
	client, err := asc.NewClientWithHTTPClient("KEY123", "ISS456", keyPath, httpClient)
	if err != nil {
		t.Fatalf("NewClientWithHTTPClient() error: %v", err)
	}
	return client
}

func validValidateSubscriptionsFixture() validateSubscriptionsFixture {
	return validateSubscriptionsFixture{
		groups: `{"data":[{"type":"subscriptionGroups","id":"group-1","attributes":{"referenceName":"Group"}}]}`,
		subscriptionsByGroup: map[string]string{
			"group-1": `{"data":[{"type":"subscriptions","id":"sub-1","attributes":{"name":"Monthly","productId":"com.example.monthly","state":"APPROVED"}}]}`,
		},
	}
}

func TestValidateSubscriptionsRequiresApp(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "subscriptions"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--app is required") {
		t.Fatalf("expected --app required error, got %q", stderr)
	}
}

func TestValidateSubscriptionsOutputsJSONAndTable(t *testing.T) {
	fixture := validValidateSubscriptionsFixture()
	client := newValidateSubscriptionsClient(t, fixture)
	restore := validate.SetClientFactory(func() (*asc.Client, error) {
		return client, nil
	})
	defer restore()

	root := RootCommand("1.2.3")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "subscriptions", "--app", "app-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var report validation.SubscriptionsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if report.Summary.Errors != 0 || report.Summary.Warnings != 0 {
		t.Fatalf("expected no issues, got %+v", report.Summary)
	}

	root = RootCommand("1.2.3")
	stdout, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "subscriptions", "--app", "app-1", "--output", "table"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if !strings.Contains(stdout, "Severity") {
		t.Fatalf("expected table output to include headers, got %q", stdout)
	}
}

func TestValidateSubscriptionsWarnsAndStrictFails(t *testing.T) {
	fixture := validValidateSubscriptionsFixture()
	fixture.subscriptionsByGroup["group-1"] = `{"data":[{"type":"subscriptions","id":"sub-1","attributes":{"name":"Monthly","productId":"com.example.monthly","state":"READY_TO_SUBMIT"}}]}`

	client := newValidateSubscriptionsClient(t, fixture)
	restore := validate.SetClientFactory(func() (*asc.Client, error) {
		return client, nil
	})
	defer restore()

	root := RootCommand("1.2.3")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "subscriptions", "--app", "app-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("expected no error (warning-only), got %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var report validation.SubscriptionsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if report.Summary.Warnings == 0 {
		t.Fatalf("expected warnings, got %+v", report.Summary)
	}

	root = RootCommand("1.2.3")
	var runErr error
	stdout, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "subscriptions", "--app", "app-1", "--strict"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatalf("expected error with --strict")
	}
	if _, ok := errors.AsType[ReportedError](runErr); !ok {
		t.Fatalf("expected ReportedError, got %v", runErr)
	}

	var strictReport validation.SubscriptionsReport
	if err := json.Unmarshal([]byte(stdout), &strictReport); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	found := false
	for _, check := range strictReport.Checks {
		if check.ID == "subscriptions.review_readiness.needs_attention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected subscriptions.review_readiness.needs_attention check, got %+v", strictReport.Checks)
	}
}
