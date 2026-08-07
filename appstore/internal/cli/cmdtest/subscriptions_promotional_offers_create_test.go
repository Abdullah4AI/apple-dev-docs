package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestSubscriptionsPromotionalOffersCreateBuildsInlinePrices(t *testing.T) {
	tests := []struct {
		name                 string
		mode                 string
		prices               string
		wantReferenceID      string
		wantInline           bool
		wantTerritory        string
		wantPricePoint       string
		wantPricePointLinked bool
		wantSecondTerritory  string
	}{
		{name: "paid compound", mode: "pay_as_you_go", prices: "United States:pp-us", wantInline: true, wantTerritory: "USA", wantPricePoint: "pp-us", wantPricePointLinked: true},
		{name: "compound and territory-only inline", mode: "pay_as_you_go", prices: "US:pp-us,France", wantInline: true, wantTerritory: "USA", wantPricePoint: "pp-us", wantPricePointLinked: true, wantSecondTerritory: "FRA"},
		{name: "free trial territory only", mode: "free_trial", prices: "Germany", wantInline: true, wantTerritory: "DEU"},
		{name: "paid territory only", mode: "pay_up_front", prices: "France", wantInline: true, wantTerritory: "FRA"},
		{name: "free trial compound", mode: "free_trial", prices: "US:pp-us", wantInline: true, wantTerritory: "USA", wantPricePoint: "pp-us", wantPricePointLinked: true},
		{name: "legacy bare price ID", mode: "pay_up_front", prices: "price-legacy", wantReferenceID: "price-legacy"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPost || req.URL.Path != "/v1/subscriptionPromotionalOffers" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				data := payload["data"].(map[string]any)
				priceRefs := data["relationships"].(map[string]any)["prices"].(map[string]any)["data"].([]any)
				wantPriceCount := 1
				if test.wantSecondTerritory != "" {
					wantPriceCount = 2
				}
				if len(priceRefs) != wantPriceCount {
					t.Fatalf("expected %d price linkage(s), got refs=%#v", wantPriceCount, priceRefs)
				}
				priceRef := priceRefs[0].(map[string]any)
				if !test.wantInline {
					if priceRef["id"] != test.wantReferenceID || priceRef["type"] != "subscriptionPromotionalOfferPrices" {
						t.Fatalf("unexpected legacy price linkage: %#v", priceRef)
					}
					if _, ok := payload["included"]; ok {
						t.Fatalf("legacy price linkage must not emit included resources: %#v", payload)
					}
					writePromotionalOfferCreateResponse(t, w)
					return
				}

				included, ok := payload["included"].([]any)
				if !ok || len(included) != wantPriceCount {
					t.Fatalf("expected %d included resource(s), got %#v", wantPriceCount, payload["included"])
				}
				includedPrice := included[0].(map[string]any)
				if priceRef["id"] != includedPrice["id"] || priceRef["type"] != "subscriptionPromotionalOfferPrices" {
					t.Fatalf("price linkage does not match included resource: ref=%#v included=%#v", priceRef, includedPrice)
				}
				relationships := includedPrice["relationships"].(map[string]any)
				territory := relationships["territory"].(map[string]any)["data"].(map[string]any)
				if territory["id"] != test.wantTerritory {
					t.Fatalf("expected territory %s, got %#v", test.wantTerritory, territory)
				}
				pricePoint, hasPricePoint := relationships["subscriptionPricePoint"]
				if hasPricePoint != test.wantPricePointLinked {
					t.Fatalf("subscriptionPricePoint presence = %t, want %t", hasPricePoint, test.wantPricePointLinked)
				}
				if hasPricePoint {
					pricePointID := pricePoint.(map[string]any)["data"].(map[string]any)["id"]
					if pricePointID != test.wantPricePoint {
						t.Fatalf("expected price point %s, got %#v", test.wantPricePoint, pricePointID)
					}
				}
				if test.wantSecondTerritory != "" {
					secondRef := priceRefs[1].(map[string]any)
					secondIncluded := included[1].(map[string]any)
					if secondRef["id"] != secondIncluded["id"] {
						t.Fatalf("second price linkage does not match included resource: ref=%#v included=%#v", secondRef, secondIncluded)
					}
					secondRelationships := secondIncluded["relationships"].(map[string]any)
					secondTerritory := secondRelationships["territory"].(map[string]any)["data"].(map[string]any)
					if secondTerritory["id"] != test.wantSecondTerritory {
						t.Fatalf("expected second territory %s, got %#v", test.wantSecondTerritory, secondTerritory)
					}
					if _, ok := secondRelationships["subscriptionPricePoint"]; ok {
						t.Fatalf("second territory-only price must not include a price point: %#v", secondRelationships)
					}
				}
				writePromotionalOfferCreateResponse(t, w)
			}))
			t.Cleanup(server.Close)

			serverHost := strings.TrimPrefix(server.URL, "http://")
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				cloned := req.Clone(req.Context())
				cloned.URL.Scheme = "http"
				cloned.URL.Host = serverHost
				return server.Client().Transport.RoundTrip(cloned)
			})
			client, err := asc.NewClientWithHTTPClient(
				os.Getenv("ASC_KEY_ID"),
				os.Getenv("ASC_ISSUER_ID"),
				os.Getenv("ASC_PRIVATE_KEY_PATH"),
				&http.Client{Transport: transport},
			)
			if err != nil {
				t.Fatalf("create test client: %v", err)
			}
			t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
				return client, nil
			}))

			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse([]string{
					"subscriptions", "offers", "promotional", "create",
					"--subscription-id", "8000000001",
					"--offer-code", "SPRING",
					"--name", "Spring",
					"--offer-duration", "one_month",
					"--offer-mode", test.mode,
					"--number-of-periods", "1",
					"--prices", test.prices,
				}); err != nil {
					t.Fatalf("parse: %v", err)
				}
				if err := root.Run(context.Background()); err != nil {
					t.Fatalf("run: %v", err)
				}
			})
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			var output struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(stdout), &output); err != nil || output.Data.ID != "promo-1" {
				t.Fatalf("unexpected stdout %q: %v", stdout, err)
			}
		})
	}
}

func writePromotionalOfferCreateResponse(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := io.WriteString(w, `{"data":{"type":"subscriptionPromotionalOffers","id":"promo-1"}}`); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func TestSubscriptionsPromotionalOffersCreateRejectsInvalidPriceShapeBeforeAuth(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		prices  string
		wantErr string
	}{
		{name: "mixed inline and legacy", mode: "PAY_UP_FRONT", prices: "US:price-point-1,price-legacy", wantErr: "must not mix"},
		{name: "mixed territory-only and legacy", mode: "PAY_UP_FRONT", prices: "US,price-legacy", wantErr: "must not mix"},
		{name: "mixed legacy and territory-only", mode: "PAY_UP_FRONT", prices: "price-legacy,US", wantErr: "must not mix"},
		{name: "inline missing price point", mode: "FREE_TRIAL", prices: "US:", wantErr: "TERRITORY:PRICE_POINT_ID"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			var runErr error
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse([]string{
					"subscriptions", "offers", "promotional", "create",
					"--subscription-id", "8000000001", "--offer-code", "SPRING", "--name", "Spring",
					"--offer-duration", "ONE_MONTH", "--offer-mode", test.mode,
					"--number-of-periods", "1", "--prices", test.prices,
				}); err != nil {
					t.Fatalf("parse: %v", err)
				}
				runErr = root.Run(context.Background())
			})
			if !errors.Is(runErr, flag.ErrHelp) {
				t.Fatalf("expected usage error, got %v", runErr)
			}
			if got := rootcmd.ExitCodeFromError(runErr); got != rootcmd.ExitUsage {
				t.Fatalf("exit code = %d, want %d", got, rootcmd.ExitUsage)
			}
			if !strings.Contains(stderr, test.wantErr) || stdout != "" {
				t.Fatalf("stdout=%q stderr=%q, want stderr containing %q", stdout, stderr, test.wantErr)
			}
		})
	}
}
