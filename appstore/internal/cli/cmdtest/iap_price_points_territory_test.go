package cmdtest

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestIAPPricePointsListNormalizesTerritory(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v2/inAppPurchases/9000000001/pricePoints" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		if got := req.URL.Query().Get("filter[territory]"); got != "FRA" {
			t.Fatalf("expected normalized filter[territory]=FRA, got %q", got)
		}
		return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"inAppPurchasePricePoints","id":"pp-1","attributes":{"customerPrice":"0.99","proceeds":"0.70"}}],"links":{"next":""}}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	if err := root.Parse([]string{
		"iap", "pricing", "price-points", "list",
		"--iap-id", "9000000001",
		"--territory", "France",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := root.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
}
