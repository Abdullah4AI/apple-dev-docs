package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestIAPAvailabilitySetNormalizesTerritories(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/inAppPurchaseAvailabilities" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}

		var payload struct {
			Data struct {
				Relationships struct {
					InAppPurchase struct {
						Data struct {
							ID string `json:"id"`
						} `json:"data"`
					} `json:"inAppPurchase"`
					AvailableTerritories struct {
						Data []struct {
							ID string `json:"id"`
						} `json:"data"`
					} `json:"availableTerritories"`
				} `json:"relationships"`
			} `json:"data"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request payload: %v", err)
		}
		if got := payload.Data.Relationships.InAppPurchase.Data.ID; got != "8000000001" {
			t.Fatalf("expected iap id 8000000001, got %q", got)
		}
		if len(payload.Data.Relationships.AvailableTerritories.Data) != 2 {
			t.Fatalf("expected two territories, got %+v", payload.Data.Relationships.AvailableTerritories.Data)
		}
		if got := payload.Data.Relationships.AvailableTerritories.Data[0].ID; got != "USA" {
			t.Fatalf("expected first territory USA, got %q", got)
		}
		if got := payload.Data.Relationships.AvailableTerritories.Data[1].ID; got != "FRA" {
			t.Fatalf("expected second territory FRA, got %q", got)
		}

		return jsonHTTPResponse(http.StatusCreated, `{"data":{"type":"inAppPurchaseAvailabilities","id":"avail-1","attributes":{"availableInNewTerritories":false}}}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	if err := root.Parse([]string{
		"iap", "pricing", "availability", "set",
		"--iap-id", "8000000001",
		"--territories", "US,France",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := root.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
}
