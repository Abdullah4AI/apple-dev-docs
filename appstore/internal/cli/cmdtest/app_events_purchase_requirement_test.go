package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAppEventsCreateNormalizesPurchaseRequirement(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appEvents" {
			t.Fatalf("expected /v1/appEvents path, got %s", req.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		data, ok := payload["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data object, got %T", payload["data"])
		}
		attrs, ok := data["attributes"].(map[string]any)
		if !ok {
			t.Fatalf("expected attributes object, got %T", data["attributes"])
		}

		if attrs["purchaseRequirement"] != "NO_COST_ASSOCIATED" {
			t.Fatalf("expected purchaseRequirement NO_COST_ASSOCIATED, got %v", attrs["purchaseRequirement"])
		}

		body := `{"data":{"type":"appEvents","id":"event-1","attributes":{"referenceName":"Launch","badge":"CHALLENGE"}}}`
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"app-events", "create",
			"--app", "APP_ID",
			"--name", "Launch",
			"--event-type", "CHALLENGE",
			"--purchase-requirement", "noCostAssociated",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"event-1"`) {
		t.Fatalf("expected created event output, got %q", stdout)
	}
}

func TestAppEventsCreateRejectsInvalidPurchaseRequirement(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"app-events", "create",
			"--app", "APP_ID",
			"--name", "Launch",
			"--event-type", "CHALLENGE",
			"--purchase-requirement", "free",
		}); err != nil {
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
	if !strings.Contains(stderr, "Error: --purchase-requirement currently supports only: NO_COST_ASSOCIATED") {
		t.Fatalf("expected invalid purchase requirement error, got %q", stderr)
	}
}

func TestAppEventsUpdateRejectsKnownUnsupportedPurchaseRequirement(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"app-events", "update",
			"--event-id", "EVENT_ID",
			"--purchase-requirement", "IAP_REQUIRED",
		}); err != nil {
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
	if !strings.Contains(stderr, "known 500 UNEXPECTED_ERROR") {
		t.Fatalf("expected known Apple 500 warning, got %q", stderr)
	}
}
