package cmdtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

func TestPricingAvailabilityRemoveFromSaleUpdatesAndVerifiesTerritories(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	var mu sync.Mutex
	states := map[string]bool{"USA": true, "FRA": false}
	var patches atomic.Int32

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-1","attributes":{"availableInNewTerritories":true}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-1/territoryAvailabilities":
			mu.Lock()
			defer mu.Unlock()
			return territoryAvailabilityResponse(t, states), nil
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/territoryAvailabilities/ta-usa":
			var payload struct {
				Data struct {
					ID         string `json:"id"`
					Attributes struct {
						Available *bool `json:"available"`
					} `json:"attributes"`
				} `json:"data"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Errorf("decode PATCH: %v", err)
				return nil, fmt.Errorf("decode PATCH: %w", err)
			}
			if payload.Data.ID != "ta-usa" || payload.Data.Attributes.Available == nil || *payload.Data.Attributes.Available {
				t.Errorf("unexpected PATCH payload: %+v", payload)
				return nil, fmt.Errorf("unexpected PATCH payload")
			}
			patches.Add(1)
			mu.Lock()
			states["USA"] = false
			mu.Unlock()
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"territoryAvailabilities","id":"ta-usa","attributes":{"available":false}}}`), nil
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"pricing", "availability", "remove-from-sale", "--app", "app-1", "--confirm"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if got := patches.Load(); got != 1 {
		t.Fatalf("PATCH count = %d, want 1", got)
	}
	var result struct {
		AppID                          string   `json:"appId"`
		AvailabilityID                 string   `json:"availabilityId"`
		Status                         string   `json:"status"`
		AvailableInNewTerritories      bool     `json:"availableInNewTerritories"`
		TotalTerritories               int      `json:"totalTerritories"`
		UpdatedTerritories             int      `json:"updatedTerritories"`
		AlreadyUnavailableTerritories  int      `json:"alreadyUnavailableTerritories"`
		VerifiedUnavailableTerritories int      `json:"verifiedUnavailableTerritories"`
		FailedTerritories              []string `json:"failedTerritories"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode result %q: %v", stdout, err)
	}
	if result.AppID != "app-1" || result.AvailabilityID != "availability-1" || result.Status != "removedFromSale" {
		t.Fatalf("unexpected identity result: %+v", result)
	}
	if !result.AvailableInNewTerritories {
		t.Fatalf("expected preserved new-territory policy, got %+v", result)
	}
	if result.TotalTerritories != 2 || result.UpdatedTerritories != 1 || result.AlreadyUnavailableTerritories != 1 || result.VerifiedUnavailableTerritories != 2 || len(result.FailedTerritories) != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if !strings.Contains(stderr, "preserved availableInNewTerritories=true") {
		t.Fatalf("expected preserved-policy caveat, got %q", stderr)
	}
}

func TestPricingAvailabilityRemoveFromSaleRequiresConfirmWithUsageExit(t *testing.T) {
	stdout, stderr := captureOutput(t, func() {
		if code := rootcmd.Run([]string{"pricing", "availability", "remove-from-sale", "--app", "app-1"}, "1.2.3"); code != rootcmd.ExitUsage {
			t.Fatalf("exit code = %d, want %d", code, rootcmd.ExitUsage)
		}
	})
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--confirm is required") {
		t.Fatalf("expected confirmation diagnostic, got %q", stderr)
	}
}

func TestPricingAvailabilityRemoveFromSaleOutputFormats(t *testing.T) {
	setupAuth(t)
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-1","attributes":{"availableInNewTerritories":false}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-1/territoryAvailabilities":
			return territoryAvailabilityResponse(t, map[string]bool{"USA": false, "FRA": false}), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	tests := []struct {
		format string
		want   string
	}{
		{format: "table", want: "Availability ID"},
		{format: "markdown", want: "| Field"},
	}
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			stdout, _ := captureOutput(t, func() {
				if err := root.Parse([]string{"pricing", "availability", "remove-from-sale", "--app", "app-1", "--confirm", "--output", test.format}); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				if err := root.Run(context.Background()); err != nil {
					t.Fatalf("run error: %v", err)
				}
			})
			if !strings.Contains(stdout, test.want) {
				t.Fatalf("%s output missing %q: %s", test.format, test.want, stdout)
			}
		})
	}
}

func TestPricingAvailabilityRemoveFromSaleNoOp(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	var patches atomic.Int32

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-1","attributes":{"availableInNewTerritories":false}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-1/territoryAvailabilities":
			return territoryAvailabilityResponse(t, map[string]bool{"USA": false, "FRA": false}), nil
		case req.Method == http.MethodPatch:
			patches.Add(1)
			return nil, fmt.Errorf("unexpected PATCH %s", req.URL.Path)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"pricing", "availability", "remove-from-sale", "--app", "app-1", "--confirm"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if got := patches.Load(); got != 0 {
		t.Fatalf("PATCH count = %d, want 0", got)
	}
	if !strings.Contains(stdout, `"updatedTerritories":0`) || !strings.Contains(stdout, `"alreadyUnavailableTerritories":2`) {
		t.Fatalf("unexpected no-op result: %s", stdout)
	}
}

func TestPricingAvailabilityRemoveFromSaleContinuesAfterPartialFailure(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	var mu sync.Mutex
	states := map[string]bool{"USA": true, "FRA": true}
	var patches atomic.Int32
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-1","attributes":{"availableInNewTerritories":false}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-1/territoryAvailabilities":
			mu.Lock()
			defer mu.Unlock()
			return territoryAvailabilityResponse(t, states), nil
		case req.Method == http.MethodPatch && strings.HasPrefix(req.URL.Path, "/v1/territoryAvailabilities/ta-"):
			patches.Add(1)
			territory := strings.ToUpper(strings.TrimPrefix(req.URL.Path, "/v1/territoryAvailabilities/ta-"))
			if territory == "FRA" {
				return jsonHTTPResponse(http.StatusUnprocessableEntity, `{"errors":[{"status":"422","code":"ENTITY_ERROR","title":"Invalid"}]}`), nil
			}
			mu.Lock()
			states[territory] = false
			mu.Unlock()
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"territoryAvailabilities","id":"ta-usa","attributes":{"available":false}}}`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"pricing", "availability", "remove-from-sale", "--app", "app-1", "--confirm"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected partial failure")
		}
		if !strings.Contains(err.Error(), "updated 1, skipped 0, failed 1 (FRA)") {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := rootcmd.ExitCodeFromError(err); got != rootcmd.ExitError {
			t.Fatalf("exit code = %d, want %d", got, rootcmd.ExitError)
		}
	})
	if got := patches.Load(); got != 2 {
		t.Fatalf("PATCH count = %d, want 2", got)
	}
	var result struct {
		Status                         string   `json:"status"`
		TotalTerritories               int      `json:"totalTerritories"`
		UpdatedTerritories             int      `json:"updatedTerritories"`
		VerifiedUnavailableTerritories int      `json:"verifiedUnavailableTerritories"`
		FailedTerritories              []string `json:"failedTerritories"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode partial-failure result %q: %v", stdout, err)
	}
	if result.Status != "partialFailure" ||
		result.TotalTerritories != 2 ||
		result.UpdatedTerritories != 1 ||
		result.VerifiedUnavailableTerritories != 1 ||
		len(result.FailedTerritories) != 1 ||
		result.FailedTerritories[0] != "FRA" {
		t.Fatalf("unexpected partial-failure result: %+v", result)
	}
}

func TestPricingAvailabilityRemoveFromSaleFinalReadbackIncludesInitiallyUnavailableTerritories(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	var territoryReads atomic.Int32
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/appAvailabilityV2":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appAvailabilities","id":"availability-1","attributes":{"availableInNewTerritories":false}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v2/appAvailabilities/availability-1/territoryAvailabilities":
			if territoryReads.Add(1) == 1 {
				return territoryAvailabilityResponse(t, map[string]bool{"USA": true, "FRA": false}), nil
			}
			return territoryAvailabilityResponse(t, map[string]bool{"USA": false, "FRA": true}), nil
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/territoryAvailabilities/ta-usa":
			return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"territoryAvailabilities","id":"ta-usa","attributes":{"available":false}}}`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	_, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"pricing", "availability", "remove-from-sale", "--app", "app-1", "--confirm"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected final-readback failure")
		}
		if !strings.Contains(err.Error(), "updated 1, skipped 1, failed 1 (FRA)") || !strings.Contains(err.Error(), "state changed during verification") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func territoryAvailabilityResponse(t *testing.T, states map[string]bool) *http.Response {
	t.Helper()
	territories := []string{"USA", "FRA"}
	data := make([]map[string]any, 0, len(territories))
	for _, territory := range territories {
		data = append(data, map[string]any{
			"type":       "territoryAvailabilities",
			"id":         "ta-" + strings.ToLower(territory),
			"attributes": map[string]any{"available": states[territory]},
			"relationships": map[string]any{
				"territory": map[string]any{"data": map[string]any{"type": "territories", "id": territory}},
			},
		})
	}
	body, err := json.Marshal(map[string]any{"data": data, "links": map[string]string{"next": ""}})
	if err != nil {
		t.Fatalf("marshal territory response: %v", err)
	}
	return jsonHTTPResponse(http.StatusOK, string(body))
}
