package cmdtest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	pricingcli "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/pricing"
)

func TestPricingAvailabilityViewMissingRecordReturnsNotFound(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/apps/app-1/appAvailabilityV2" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"status":"404","code":"NOT_FOUND","title":"not found","detail":"missing"}]}`))
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = serverURL.Scheme
		cloned.URL.Host = serverURL.Host
		return server.Client().Transport.RoundTrip(cloned)
	})
	client, err := asc.NewClientWithHTTPClient(
		"TEST_KEY",
		"TEST_ISSUER",
		os.Getenv("ASC_PRIVATE_KEY_PATH"),
		&http.Client{Transport: transport},
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	restore := pricingcli.SetAvailabilityClientFactory(func() (*asc.Client, error) {
		return client, nil
	})
	t.Cleanup(restore)

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"pricing", "availability", "view", "--app", "app-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if runErr == nil {
		t.Fatal("expected not-found error")
	}
	if !errors.Is(runErr, asc.ErrNotFound) {
		t.Fatalf("expected asc.ErrNotFound, got %v", runErr)
	}
	var apiErr *asc.APIError
	if !errors.As(runErr, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected wrapped 404 API error, got %v", runErr)
	}
	if got := cmd.ExitCodeFromError(runErr); got != cmd.ExitNotFound {
		t.Fatalf("expected exit code %d, got %d", cmd.ExitNotFound, got)
	}
	if !strings.Contains(runErr.Error(), `pricing availability view: app availability not found for app "app-1"`) {
		t.Fatalf("expected missing-availability message, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}
