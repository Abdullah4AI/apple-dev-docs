package cmdtest

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestWinBackOffersPricesNormalizesTerritories(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", req.Method)
		}
		if req.URL.Path != "/v1/winBackOffers/offer-1/prices" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		if got := req.URL.Query().Get("filter[territory]"); got != "USA,FRA" {
			t.Fatalf("expected normalized filter[territory]=USA,FRA, got %q", got)
		}
		return jsonResponse(http.StatusOK, `{"data":[]}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "offers", "win-back", "prices",
			"--id", "offer-1",
			"--territory", "US,France",
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
	if stdout == "" {
		t.Fatal("expected JSON output")
	}
}

func TestWinBackOffersPricesNormalizesCommaContainingTerritoryNames(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", req.Method)
		}
		if req.URL.Path != "/v1/winBackOffers/offer-1/prices" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		if got := req.URL.Query().Get("filter[territory]"); got != "MDA,BOL" {
			t.Fatalf("expected normalized filter[territory]=MDA,BOL, got %q", got)
		}
		return jsonResponse(http.StatusOK, `{"data":[]}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "offers", "win-back", "prices",
			"--id", "offer-1",
			"--territory", "Moldova, Republic of,Bolivia, Plurinational State of",
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
	if stdout == "" {
		t.Fatal("expected JSON output")
	}
}
