package cmdtest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPricingAvailabilityEditNormalizesTerritories(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	patched := map[string]bool{}
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-1","attributes":{"availableInNewTerritories":false}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-1/territoryAvailabilities":
			return jsonHTTPResponse(http.StatusOK, `{
				"data":[
					{"type":"territoryAvailabilities","id":"ta-us","relationships":{"territory":{"data":{"type":"territories","id":"USA"}}}},
					{"type":"territoryAvailabilities","id":"ta-fr","relationships":{"territory":{"data":{"type":"territories","id":"FRA"}}}}
				],
				"links":{"next":""}
			}`), nil
		case req.Method == http.MethodPatch && strings.HasPrefix(req.URL.Path, "/v1/territoryAvailabilities/"):
			patched[strings.TrimPrefix(req.URL.Path, "/v1/territoryAvailabilities/")] = true
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"territoryAvailabilities","id":"patched","attributes":{"available":true}}}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"pricing", "availability", "edit",
			"--app", "app-1",
			"--territory", "US,France",
			"--available", "true",
			"--available-in-new-territories", "false",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" && !strings.Contains(stderr, "Updating availability for") {
		t.Fatalf("unexpected stderr %q", stderr)
	}
	if stdout == "" {
		t.Fatal("expected command output")
	}
	if !patched["ta-us"] || !patched["ta-fr"] {
		t.Fatalf("expected normalized territories to be patched, got %#v", patched)
	}
}
