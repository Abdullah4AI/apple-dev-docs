package cmdtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// Regression coverage for https://github.com/rorkai/App-Store-Connect-CLI/issues/1948:
// FREE_TRIAL win-back offers must be creatable without --price, because the
// API rejects subscriptionPricePoint relationships on free offers.

func winBackFreeTrialCreateArgs(extra ...string) []string {
	args := []string{
		"subscriptions", "offers", "win-back", "create",
		"--subscription-id", "1234567890",
		"--reference-name", "Yearly winback",
		"--offer-id", "yearly_winback_1mo",
		"--duration", "ONE_MONTH",
		"--offer-mode", "FREE_TRIAL",
		"--period-count", "1",
		"--eligibility-paid-months", "1",
		"--eligibility-last-subscribed-min", "1",
		"--eligibility-last-subscribed-max", "12",
		"--start-date", "2026-08-11",
		"--priority", "HIGH",
	}
	return append(args, extra...)
}

func TestWinBackOffersCreateFreeTrialSendsTerritoryOnlyPrices(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	setupAuth(t)

	requestBodies := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", req.Method)
			http.Error(w, "unexpected method", http.StatusBadRequest)
			return
		}
		if req.URL.Path != "/v1/winBackOffers" {
			t.Errorf("unexpected path %q", req.URL.Path)
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}
		if req.URL.RawQuery != "" {
			t.Errorf("unexpected query %q", req.URL.RawQuery)
			http.Error(w, "unexpected query", http.StatusBadRequest)
			return
		}
		authorization := req.Header.Get("Authorization")
		token, ok := strings.CutPrefix(authorization, "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			t.Errorf("Authorization = %q, want nonempty Bearer token", authorization)
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		select {
		case requestBodies <- body:
		default:
			t.Errorf("unexpected extra request: %s %s", req.Method, req.URL.String())
			http.Error(w, "unexpected extra request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"type":"winBackOffers","id":"offer-1","attributes":{"referenceName":"Yearly winback","offerId":"yearly_winback_1mo"}}}`)
	}))
	t.Cleanup(server.Close)

	client := newWinBackOfferTestClient(t, server)
	clientFactoryCalls := 0
	t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
		clientFactoryCalls++
		return client, nil
	}))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse(winBackFreeTrialCreateArgs("--territory", "US,France")); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var response asc.WinBackOfferResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("decode stdout JSON: %v; stdout=%q", err, stdout)
	}
	if response.Data.Type != asc.ResourceTypeWinBackOffers || response.Data.ID != "offer-1" {
		t.Fatalf("response data = %+v, want winBackOffers offer-1", response.Data)
	}
	body := <-requestBodies
	var payload asc.WinBackOfferCreateRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode request body: %v; body=%s", err, body)
	}
	if payload.Data.Attributes.OfferMode != asc.SubscriptionOfferModeFreeTrial {
		t.Fatalf("offerMode = %q, want FREE_TRIAL", payload.Data.Attributes.OfferMode)
	}
	if bytes.Contains(body, []byte(`"subscriptionPricePoint"`)) {
		t.Fatalf("FREE_TRIAL payload must omit subscriptionPricePoint: %s", body)
	}
	if got := len(payload.Data.Relationships.Prices.Data); got != 2 {
		t.Fatalf("price linkages = %d, want 2", got)
	}
	if got := len(payload.Included); got != 2 {
		t.Fatalf("included prices = %d, want 2", got)
	}
	wantTerritories := []string{"USA", "FRA"}
	for i, included := range payload.Included {
		wantID := fmt.Sprintf("${price-%d}", i+1)
		linkage := payload.Data.Relationships.Prices.Data[i]
		if linkage.Type != asc.ResourceTypeWinBackOfferPrices || linkage.ID != wantID {
			t.Fatalf("price linkage[%d] = %+v, want winBackOfferPrices %s", i, linkage, wantID)
		}
		if included.Type != asc.ResourceTypeWinBackOfferPrices || included.ID != wantID {
			t.Fatalf("included[%d] = %s %s, want winBackOfferPrices %s", i, included.Type, included.ID, wantID)
		}
		if included.Relationships == nil {
			t.Fatalf("included[%d] is missing relationships: %s", i, body)
		}
		if included.Relationships.SubscriptionPricePoint != nil {
			t.Fatalf("included[%d] must not carry subscriptionPricePoint: %s", i, body)
		}
		territory := included.Relationships.Territory.Data
		if territory.Type != asc.ResourceTypeTerritories || territory.ID != wantTerritories[i] {
			t.Fatalf("included[%d] territory = %+v, want %s", i, territory, wantTerritories[i])
		}
	}
	if clientFactoryCalls != 1 {
		t.Fatalf("client factory calls = %d, want 1", clientFactoryCalls)
	}
	select {
	case extra := <-requestBodies:
		t.Fatalf("unexpected extra request body: %s", extra)
	default:
	}
}

func newWinBackOfferTestClient(t *testing.T, server *httptest.Server) *asc.Client {
	t.Helper()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.Scheme + "://" + req.URL.Host; got != asc.BaseURL {
			t.Fatalf("request origin = %s, want %s", got, asc.BaseURL)
		}
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = serverURL.Scheme
		cloned.URL.Host = serverURL.Host
		cloned.Host = serverURL.Host
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
	return client
}

func TestWinBackOffersCreateFreeTrialRejectsPrice(t *testing.T) {
	for _, value := range []string{
		"eyJzIjoiMTIzNCIsInQiOiJVU0EiLCJwIjoiMTAwOTkifQ",
		"",
		",,",
	} {
		t.Run(fmt.Sprintf("price=%q", value), func(t *testing.T) {
			t.Setenv("ASC_APP_ID", "")
			assertWinBackOfferUsageBeforeClient(t, winBackFreeTrialCreateArgs(
				"--territory", "USA",
				"--price", value,
			), "--price is not supported when --offer-mode is FREE_TRIAL")
		})
	}
}

func TestWinBackOffersCreateFreeTrialRequiresTerritory(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	assertWinBackOfferUsageBeforeClient(t, winBackFreeTrialCreateArgs(),
		"--territory is required when --offer-mode is FREE_TRIAL")
}

func TestWinBackOffersCreatePaidRejectsTerritory(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	args := winBackFreeTrialCreateArgs("--territory", "USA")
	for i, arg := range args {
		if arg == "FREE_TRIAL" {
			args[i] = "PAY_AS_YOU_GO"
		}
	}
	assertWinBackOfferUsageBeforeClient(t, args,
		"--territory is only supported when --offer-mode is FREE_TRIAL")
}

func assertWinBackOfferUsageBeforeClient(t *testing.T, args []string, wantErr string) {
	t.Helper()
	t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
		return nil, fmt.Errorf("unexpected client creation")
	}))
	assertUsageExit(t, args, wantErr)
}
