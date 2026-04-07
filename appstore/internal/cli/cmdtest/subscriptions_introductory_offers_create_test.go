package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestSubscriptionsIntroductoryOffersCreateNormalizesTerritory(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/subscriptionIntroductoryOffers" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}

		var payload struct {
			Data struct {
				Relationships struct {
					Territory struct {
						Data struct {
							ID string `json:"id"`
						} `json:"data"`
					} `json:"territory"`
				} `json:"relationships"`
			} `json:"data"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request payload: %v", err)
		}
		if got := payload.Data.Relationships.Territory.Data.ID; got != "USA" {
			t.Fatalf("expected normalized territory USA, got %q", got)
		}

		return jsonHTTPResponse(http.StatusCreated, `{"data":{"type":"subscriptionIntroductoryOffers","id":"intro-1","attributes":{"duration":"ONE_MONTH","offerMode":"FREE_TRIAL","numberOfPeriods":1}}}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	if err := root.Parse([]string{
		"subscriptions", "offers", "introductory", "create",
		"--subscription-id", "8000000001",
		"--offer-duration", "ONE_MONTH",
		"--offer-mode", "FREE_TRIAL",
		"--number-of-periods", "1",
		"--territory", "US",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := root.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
}
