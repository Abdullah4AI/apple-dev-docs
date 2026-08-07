package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestPricingPricePointViewPreservesStableCollectionResponse(t *testing.T) {
	setupAuth(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/v3/appPricePoints/price-point-1" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"type":"appPricePoints","id":"price-point-1","attributes":{"customerPrice":"0.99","proceeds":"0.70"}},"included":[{"type":"territories","id":"USA","attributes":{"currency":"USD"}}],"links":{"self":"https://api.appstoreconnect.apple.com/v3/appPricePoints/price-point-1"}}`))
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = serverURL.Scheme
		cloned.URL.Host = serverURL.Host
		return server.Client().Transport.RoundTrip(cloned)
	})
	client, err := asc.NewClientWithHTTPClient(
		os.Getenv("ASC_KEY_ID"),
		os.Getenv("ASC_ISSUER_ID"),
		os.Getenv("ASC_PRIVATE_KEY_PATH"),
		&http.Client{Transport: transport},
	)
	if err != nil {
		t.Fatalf("create pricing test client: %v", err)
	}
	restoreClient := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
		return client, nil
	})
	t.Cleanup(restoreClient)

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"pricing", "price-points", "view",
			"--price-point", "price-point-1",
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("decode stdout: %v (stdout=%q)", err, stdout)
	}
	var data []map[string]any
	if err := json.Unmarshal(envelope["data"], &data); err != nil {
		t.Fatalf("expected one-element data array, got %s: %v", envelope["data"], err)
	}
	if len(data) != 1 || data[0]["id"] != "price-point-1" {
		t.Fatalf("expected price-point-1, got %#v", data)
	}
	var included []map[string]any
	if err := json.Unmarshal(envelope["included"], &included); err != nil {
		t.Fatalf("decode included: %v", err)
	}
	if len(included) != 1 || included[0]["id"] != "USA" {
		t.Fatalf("unexpected included resources: %#v", included)
	}
	var links map[string]any
	if err := json.Unmarshal(envelope["links"], &links); err != nil {
		t.Fatalf("decode links: %v", err)
	}
	if links["self"] != "https://api.appstoreconnect.apple.com/v3/appPricePoints/price-point-1" {
		t.Fatalf("unexpected self link: %v", links["self"])
	}
}
