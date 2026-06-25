package subscriptions

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestParseSubscriptionIntroductoryOffersImportCSVHeader_StripsUTF8BOM(t *testing.T) {
	got, err := parseSubscriptionIntroductoryOffersImportCSVHeader([]string{"\ufeffterritory", "offer_mode"})
	if err != nil {
		t.Fatalf("parseSubscriptionIntroductoryOffersImportCSVHeader() error: %v", err)
	}
	if got["territory"] != 0 {
		t.Fatalf("expected territory column at index 0, got %d", got["territory"])
	}
	if got["offer_mode"] != 1 {
		t.Fatalf("expected offer_mode column at index 1, got %d", got["offer_mode"])
	}
}

func TestWriteSubscriptionIntroductoryOfferImportFailureArtifact_ReturnsWriteError(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".asc", []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	_, err := writeSubscriptionIntroductoryOfferImportFailureArtifact(&subscriptionIntroductoryOfferImportSummary{
		Failed:  1,
		Results: []subscriptionIntroductoryOfferImportResultItem{{Status: "failed"}},
	})
	if err == nil {
		t.Fatal("expected write error, got nil")
	}
}

func TestSubscriptionIntroductoryOfferImportStateMatchesImmediateUpfrontOffer(t *testing.T) {
	index := &subscriptionIntroductoryOfferImportStateIndex{
		now: time.Date(2026, time.June, 24, 12, 0, 0, 0, time.UTC),
		offers: []subscriptionIntroductoryOfferImportResolvedRow{{
			territory:       "USA",
			offerMode:       "FREE_TRIAL",
			offerDuration:   "ONE_WEEK",
			numberOfPeriods: 1,
			startDate:       "2026-06-23",
			planType:        asc.SubscriptionPlanTypeUpfront,
		}},
	}
	target := subscriptionIntroductoryOfferImportResolvedRow{
		territory:       "USA",
		offerMode:       "FREE_TRIAL",
		offerDuration:   "ONE_WEEK",
		numberOfPeriods: 1,
		planType:        asc.SubscriptionPlanTypeUpfront,
	}
	if !index.matches(target) {
		t.Fatal("expected an already-active UPFRONT offer to match an immediate target")
	}

	index.offers[0].startDate = "2026-06-25"
	if index.matches(target) {
		t.Fatal("expected a future offer not to match an immediate target")
	}

	index.offers[0].startDate = "2026-06-23"
	index.offers[0].planType = asc.SubscriptionPlanTypeMonthly
	if index.matches(target) {
		t.Fatal("expected a MONTHLY offer not to match an UPFRONT target")
	}
}

func TestSubscriptionIntroductoryOfferImportReconcilesAcrossUTCMidnight(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	originalNow := subscriptionImportNow
	t.Cleanup(func() { subscriptionImportNow = originalNow })
	current := time.Date(2026, time.June, 24, 23, 59, 0, 0, time.UTC)
	subscriptionImportNow = func() time.Time {
		return current
	}

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = introImportRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/subscriptions/sub-1/introductoryOffers" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		current = time.Date(2026, time.June, 25, 0, 1, 0, 0, time.UTC)
		body := `{"data":[{"type":"subscriptionIntroductoryOffers","id":"offer-1","attributes":{"startDate":"2026-06-25","duration":"ONE_WEEK","offerMode":"FREE_TRIAL","numberOfPeriods":1,"targetSubscriptionPlanType":"UPFRONT"},"relationships":{"territory":{"data":{"type":"territories","id":"USA"}}}}],"links":{}}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
	})
	client, err := asc.NewClientFromPEM("KEY123", "issuer", introImportTestPrivateKeyPEM(t))
	if err != nil {
		t.Fatalf("NewClientFromPEM() error: %v", err)
	}

	target := subscriptionIntroductoryOfferImportResolvedRow{
		territory: "USA", offerMode: "FREE_TRIAL", offerDuration: "ONE_WEEK", numberOfPeriods: 1, planType: asc.SubscriptionPlanTypeUpfront,
	}
	mutations := 0
	status, err := runReconciledMutation(
		context.Background(),
		func(readbackCtx context.Context) (bool, error) {
			index, err := fetchSubscriptionIntroductoryOfferImportState(readbackCtx, client, "sub-1")
			if err != nil {
				return false, err
			}
			return index.matches(target), nil
		},
		func(context.Context) error {
			mutations++
			return &asc.RetryableError{Err: errors.New("ambiguous timeout")}
		},
	)
	if err != nil {
		t.Fatalf("runReconciledMutation() error: %v", err)
	}
	if status != reconciledMutationReconciled || mutations != 1 {
		t.Fatalf("unexpected midnight recovery: status=%q mutations=%d", status, mutations)
	}
}

type introImportRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn introImportRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func introImportTestPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey() error: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}
