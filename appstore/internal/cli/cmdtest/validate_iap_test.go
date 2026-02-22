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

type validateIAPFixture struct {
	iaps string
}

func newValidateIAPClient(t *testing.T, fixture validateIAPFixture) *asc.Client {
	t.Helper()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.p8")
	writeECDSAPEM(t, keyPath)

	notFound := `{"errors":[{"code":"NOT_FOUND","title":"Not Found","detail":"resource not found"}]}`

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return jsonResponse(http.StatusMethodNotAllowed, `{"errors":[{"status":405}]}`)
		}

		switch req.URL.Path {
		case "/v1/apps/app-1/inAppPurchasesV2":
			return jsonResponse(http.StatusOK, fixture.iaps)
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

func validValidateIAPFixture() validateIAPFixture {
	return validateIAPFixture{
		iaps: `{"data":[{"type":"inAppPurchases","id":"iap-1","attributes":{"name":"Pro","productId":"com.example.pro","inAppPurchaseType":"NON_CONSUMABLE","state":"APPROVED"}}]}`,
	}
}

func TestValidateIAPRequiresApp(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "iap"}); err != nil {
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

func TestValidateIAPOutputsJSONAndTable(t *testing.T) {
	fixture := validValidateIAPFixture()
	client := newValidateIAPClient(t, fixture)
	restore := validate.SetClientFactory(func() (*asc.Client, error) {
		return client, nil
	})
	defer restore()

	root := RootCommand("1.2.3")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "iap", "--app", "app-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var report validation.IAPReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if report.Summary.Errors != 0 || report.Summary.Warnings != 0 {
		t.Fatalf("expected no issues, got %+v", report.Summary)
	}

	root = RootCommand("1.2.3")
	stdout, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "iap", "--app", "app-1", "--output", "table"}); err != nil {
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

func TestValidateIAPWarnsAndStrictFails(t *testing.T) {
	fixture := validValidateIAPFixture()
	fixture.iaps = `{"data":[{"type":"inAppPurchases","id":"iap-1","attributes":{"name":"Pro","productId":"com.example.pro","inAppPurchaseType":"NON_CONSUMABLE","state":"READY_TO_SUBMIT"}}]}`

	client := newValidateIAPClient(t, fixture)
	restore := validate.SetClientFactory(func() (*asc.Client, error) {
		return client, nil
	})
	defer restore()

	root := RootCommand("1.2.3")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "iap", "--app", "app-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("expected no error (warning-only), got %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var report validation.IAPReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if report.Summary.Warnings == 0 {
		t.Fatalf("expected warnings, got %+v", report.Summary)
	}

	root = RootCommand("1.2.3")
	var runErr error
	stdout, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"validate", "iap", "--app", "app-1", "--strict"}); err != nil {
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

	var strictReport validation.IAPReport
	if err := json.Unmarshal([]byte(stdout), &strictReport); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	found := false
	for _, check := range strictReport.Checks {
		if check.ID == "iap.review_readiness.needs_attention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected iap.review_readiness.needs_attention check, got %+v", strictReport.Checks)
	}
}
