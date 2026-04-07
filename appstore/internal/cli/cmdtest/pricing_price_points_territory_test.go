package cmdtest

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestPricingPricePointsNormalizesTerritory(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/apps/app-1/appPricePoints" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		if got := req.URL.Query().Get("filter[territory]"); got != "USA" {
			t.Fatalf("expected normalized filter[territory]=USA, got %q", got)
		}
		return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"appPricePoints","id":"pp-1","attributes":{"customerPrice":"0.99","proceeds":"0.70"}}],"links":{"next":""}}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	if err := root.Parse([]string{
		"pricing", "price-points",
		"--app", "app-1",
		"--territory", "United States",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := root.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
}
