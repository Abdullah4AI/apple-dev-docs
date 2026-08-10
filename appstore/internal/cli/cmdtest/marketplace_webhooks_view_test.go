package cmdtest

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

func TestMarketplaceWebhooksViewUsesCollectionAndPrintsSingleResource(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	requests := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/v1/marketplaceWebhooks" {
			t.Fatalf("expected collection GET, got %s %s", req.Method, req.URL.String())
		}
		if req.URL.Query().Get("limit") != "200" {
			t.Fatalf("expected limit=200, got %q", req.URL.Query().Get("limit"))
		}
		return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"marketplaceWebhooks","id":"wh-1","attributes":{"endpointUrl":"https://example.com/webhook"}}],"links":{}}`), nil
	})

	stdout, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{"marketplace", "webhooks", "view", "--webhook-id", "wh-1", "--output", "json"}, "1.2.3")
		if code != rootcmd.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d", rootcmd.ExitSuccess, code)
		}
	})

	if requests != 1 {
		t.Fatalf("expected one request, got %d", requests)
	}
	if !strings.Contains(stderr, "Warning: marketplace webhooks endpoints are deprecated in App Store Connect API.") {
		t.Fatalf("expected deprecation warning, got %q", stderr)
	}
	var output struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				EndpointURL string `json:"endpointUrl"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout=%s", err, stdout)
	}
	if output.Data.ID != "wh-1" || output.Data.Attributes.EndpointURL != "https://example.com/webhook" {
		t.Fatalf("unexpected output: %+v", output.Data)
	}
}

func TestMarketplaceWebhooksViewMissingIDReturnsUsageExitCode(t *testing.T) {
	stdout, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{"marketplace", "webhooks", "view"}, "1.2.3")
		if code != rootcmd.ExitUsage {
			t.Fatalf("expected exit code %d, got %d", rootcmd.ExitUsage, code)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --webhook-id is required") {
		t.Fatalf("expected missing-ID error, got %q", stderr)
	}
	if strings.Count(stderr, "Error: --webhook-id is required") != 1 {
		t.Fatalf("expected one missing-ID error, got %q", stderr)
	}
}

func TestMarketplaceWebhooksViewNotFoundReturnsNotFoundExitCode(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, `{"data":[],"links":{}}`), nil
	})

	stdout, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{"marketplace", "webhooks", "view", "--webhook-id", "wh-missing", "--output", "json"}, "1.2.3")
		if code != rootcmd.ExitNotFound {
			t.Fatalf("expected exit code %d, got %d", rootcmd.ExitNotFound, code)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, `marketplace webhooks view: failed to fetch: marketplace webhook "wh-missing" not found`) {
		t.Fatalf("expected contextual not-found error, got %q", stderr)
	}
}

func TestMarketplaceWebhooksViewAPIErrorPreservesExitCode(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusBadRequest, `{"errors":[{"status":"400","code":"INVALID_REQUEST","title":"Invalid Request","detail":"request rejected"}]}`), nil
	})

	stdout, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{"marketplace", "webhooks", "view", "--webhook-id", "wh-1", "--output", "json"}, "1.2.3")
		if code != rootcmd.ExitHTTPBadRequest {
			t.Fatalf("expected exit code %d, got %d", rootcmd.ExitHTTPBadRequest, code)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "marketplace webhooks view: failed to fetch: Invalid Request: request rejected") {
		t.Fatalf("expected API error, got %q", stderr)
	}
}

func TestMarketplaceWebhooksViewHelpIsRegistered(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	view := findSubcommand(root, "marketplace", "webhooks", "view")
	if view == nil {
		t.Fatal("expected marketplace webhooks view command")
	}
	if view.ShortUsage != `asc marketplace webhooks view --webhook-id "WEBHOOK_ID" [flags]` {
		t.Fatalf("unexpected help usage: %q", view.ShortUsage)
	}
	if view.FlagSet.Lookup("webhook-id") == nil || view.FlagSet.Lookup("output") == nil || view.FlagSet.Lookup("pretty") == nil {
		t.Fatalf("expected webhook-id and output flags in view help")
	}
}
