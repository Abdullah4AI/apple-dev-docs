package subscriptions

import (
	"context"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestResolveSubscriptionPriceDerivePoint(t *testing.T) {
	tests := []struct {
		name       string
		candidates []subscriptionPriceDeriveCandidate
		desired    string
		rounding   subscriptionPriceDeriveRounding
		wantID     string
		wantPrice  string
		wantErr    string
	}{
		{
			name:       "exact accepts numerically equal decimals",
			candidates: deriveCandidates("lower", "8.99", "exact", "9.00", "upper", "9.99"),
			desired:    "9",
			rounding:   subscriptionPriceDeriveExact,
			wantID:     "exact",
			wantPrice:  "9.00",
		},
		{
			name:       "exact rejects a missing amount",
			candidates: deriveCandidates("lower", "89", "upper", "99"),
			desired:    "90",
			rounding:   subscriptionPriceDeriveExact,
			wantErr:    "no exact target price point for 90",
		},
		{
			name:       "nearest chooses lower distance",
			candidates: deriveCandidates("upper", "99", "lower", "89"),
			desired:    "90",
			rounding:   subscriptionPriceDeriveNearest,
			wantID:     "lower",
			wantPrice:  "89",
		},
		{
			name:       "nearest chooses upper distance",
			candidates: deriveCandidates("lower", "89", "upper", "99"),
			desired:    "97",
			rounding:   subscriptionPriceDeriveNearest,
			wantID:     "upper",
			wantPrice:  "99",
		},
		{
			name:       "nearest tie chooses lower",
			candidates: deriveCandidates("upper", "100", "lower", "90"),
			desired:    "95",
			rounding:   subscriptionPriceDeriveNearest,
			wantID:     "lower",
			wantPrice:  "90",
		},
		{
			name:       "nearest below ladder chooses minimum",
			candidates: deriveCandidates("high", "20", "low", "10"),
			desired:    "1",
			rounding:   subscriptionPriceDeriveNearest,
			wantID:     "low",
			wantPrice:  "10",
		},
		{
			name:       "nearest above ladder chooses maximum",
			candidates: deriveCandidates("high", "20", "low", "10"),
			desired:    "30",
			rounding:   subscriptionPriceDeriveNearest,
			wantID:     "high",
			wantPrice:  "20",
		},
		{
			name:       "up chooses exact boundary",
			candidates: deriveCandidates("high", "20.00", "low", "10"),
			desired:    "20",
			rounding:   subscriptionPriceDeriveUp,
			wantID:     "high",
			wantPrice:  "20.00",
		},
		{
			name:       "up chooses smallest greater amount",
			candidates: deriveCandidates("high", "20", "low", "10", "middle", "15"),
			desired:    "12",
			rounding:   subscriptionPriceDeriveUp,
			wantID:     "middle",
			wantPrice:  "15",
		},
		{
			name:       "up fails above ladder",
			candidates: deriveCandidates("high", "20", "low", "10"),
			desired:    "21",
			rounding:   subscriptionPriceDeriveUp,
			wantErr:    "no target price point at or above 21",
		},
		{
			name:       "down chooses exact boundary",
			candidates: deriveCandidates("high", "20", "low", "10.00"),
			desired:    "10",
			rounding:   subscriptionPriceDeriveDown,
			wantID:     "low",
			wantPrice:  "10.00",
		},
		{
			name:       "down chooses largest lower amount",
			candidates: deriveCandidates("high", "20", "low", "10", "middle", "15"),
			desired:    "18",
			rounding:   subscriptionPriceDeriveDown,
			wantID:     "middle",
			wantPrice:  "15",
		},
		{
			name:       "down fails below ladder",
			candidates: deriveCandidates("high", "20", "low", "10"),
			desired:    "9",
			rounding:   subscriptionPriceDeriveDown,
			wantErr:    "no target price point at or below 9",
		},
		{
			name:       "selected duplicate amount is ambiguous",
			candidates: deriveCandidates("one", "9", "two", "9.00", "other", "10"),
			desired:    "9",
			rounding:   subscriptionPriceDeriveNearest,
			wantErr:    "multiple target price points match selected price 9",
		},
		{
			name:       "malformed candidate fails explicitly",
			candidates: deriveCandidates("valid", "9", "broken", "not-a-price"),
			desired:    "9",
			rounding:   subscriptionPriceDeriveNearest,
			wantErr:    `target price point "broken" has invalid customer price`,
		},
		{
			name:       "repeated id with conflicting prices fails explicitly",
			candidates: deriveCandidates("same", "9", "same", "10"),
			desired:    "9",
			rounding:   subscriptionPriceDeriveNearest,
			wantErr:    `target price point "same" has conflicting customer prices`,
		},
		{
			name:       "identical repeated resources are deduplicated",
			candidates: deriveCandidates("same", "9", "same", "9.00", "other", "10"),
			desired:    "9",
			rounding:   subscriptionPriceDeriveNearest,
			wantID:     "same",
			wantPrice:  "9",
		},
		{
			name:     "empty ladder fails",
			desired:  "9",
			rounding: subscriptionPriceDeriveNearest,
			wantErr:  "target price-point ladder is empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			desired, ok := new(big.Rat).SetString(test.desired)
			if !ok {
				t.Fatalf("invalid test desired price %q", test.desired)
			}
			got, err := resolveSubscriptionPriceDerivePoint(test.candidates, desired, test.rounding)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveSubscriptionPriceDerivePoint() error = %v", err)
			}
			if got.PricePointID != test.wantID || got.CustomerPrice != test.wantPrice {
				t.Fatalf("resolution = %+v, want id=%q price=%q", got, test.wantID, test.wantPrice)
			}
		})
	}
}

func TestSubscriptionPriceDeriveUsesExactDecimalMultiplication(t *testing.T) {
	source, _ := new(big.Rat).SetString("0.1")
	multiplier, _ := new(big.Rat).SetString("3")
	desired := new(big.Rat).Mul(source, multiplier)

	got, err := resolveSubscriptionPriceDerivePoint(
		deriveCandidates("exact", "0.30", "upper", "0.31"),
		desired,
		subscriptionPriceDeriveExact,
	)
	if err != nil {
		t.Fatalf("resolveSubscriptionPriceDerivePoint() error = %v", err)
	}
	if got.PricePointID != "exact" {
		t.Fatalf("expected exact decimal point, got %+v", got)
	}
}

func TestFormatSubscriptionPriceDeriveExactDecimalPreservesInputPrecision(t *testing.T) {
	value, ok := new(big.Rat).SetString("1.0000000000001")
	if !ok {
		t.Fatal("parse exact test decimal")
	}
	if got := formatSubscriptionPriceDeriveExactDecimal(value); got != "1.0000000000001" {
		t.Fatalf("exact decimal = %q, want 1.0000000000001", got)
	}
}

func TestFormatSubscriptionPriceDeriveDecimalPreservesNonzeroValues(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		precision int
		want      string
	}{
		{name: "positive below requested precision", value: "0.0000001", precision: 6, want: "0.0000001"},
		{name: "negative below requested precision", value: "-0.0000001", precision: 6, want: "-0.0000001"},
		{name: "small recurring rational", value: "1/30000000", precision: 6, want: "0.00000003"},
		{name: "exact zero", value: "0", precision: 6, want: "0"},
		{name: "ordinary rounding remains bounded", value: "1/9", precision: 6, want: "0.111111"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, ok := new(big.Rat).SetString(test.value)
			if !ok {
				t.Fatalf("parse test value %q", test.value)
			}
			if got := formatSubscriptionPriceDeriveDecimal(value, test.precision); got != test.want {
				t.Fatalf("formatted decimal = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFetchSubscriptionPriceDeriveCandidatesBatchesTerritoriesAndPaginates(t *testing.T) {
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	calls := 0
	http.DefaultTransport = resolvedPricesRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			if req.URL.Path != "/v1/subscriptions/target-sub/pricePoints" {
				t.Fatalf("first path = %q", req.URL.Path)
			}
			query := req.URL.Query()
			if query.Get("filter[territory]") != "SWE,USA" || query.Get("filter[planType]") != "UPFRONT" {
				t.Fatalf("unexpected first-page filters: %s", req.URL.RawQuery)
			}
			if query.Get("fields[subscriptionPricePoints]") != "customerPrice,territory" || query.Get("include") != "territory" || query.Get("limit") != "8000" {
				t.Fatalf("unexpected first-page sparse fields/limit: %s", req.URL.RawQuery)
			}
			return resolvedPricesJSONResponse(`{"data":[{"type":"subscriptionPricePoints","id":"pp-899","attributes":{"customerPrice":"8.99"},"relationships":{"territory":{"data":{"type":"territories","id":"USA"}}}}],"links":{"next":"/v1/subscriptions/target-sub/pricePoints?cursor=next"}}`), nil
		case 2:
			if req.URL.Path != "/v1/subscriptions/target-sub/pricePoints" || req.URL.Query().Get("cursor") != "next" {
				t.Fatalf("unexpected continuation request: %s", req.URL.String())
			}
			return resolvedPricesJSONResponse(`{"data":[{"type":"subscriptionPricePoints","id":"pp-89","attributes":{"customerPrice":"89"},"relationships":{"territory":{"data":{"type":"territories","id":"SWE"}}}}],"links":{}}`), nil
		default:
			t.Fatalf("unexpected extra request: %s", req.URL.String())
			return nil, nil
		}
	})
	client, err := asc.NewClientFromPEM("KEY123", "issuer", introImportTestPrivateKeyPEM(t))
	if err != nil {
		t.Fatalf("NewClientFromPEM() error: %v", err)
	}

	candidates, err := fetchSubscriptionPriceDeriveCandidatesByTerritory(context.Background(), client, "target-sub", []string{"USA", "SWE", "USA"})
	if err != nil {
		t.Fatalf("fetchSubscriptionPriceDeriveCandidatesByTerritory() error = %v", err)
	}
	if calls != 2 || len(candidates) != 2 || len(candidates["SWE"]) != 1 || candidates["SWE"][0].PricePointID != "pp-89" || len(candidates["USA"]) != 1 || candidates["USA"][0].PricePointID != "pp-899" {
		t.Fatalf("unexpected paginated candidates: calls=%d candidates=%+v", calls, candidates)
	}
}

func TestFetchSubscriptionPriceDeriveCandidatesRequiresTerritoryRelationship(t *testing.T) {
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = resolvedPricesRoundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return resolvedPricesJSONResponse(`{"data":[{"type":"subscriptionPricePoints","id":"pp-89","attributes":{"customerPrice":"89"}}],"links":{}}`), nil
	})
	client, err := asc.NewClientFromPEM("KEY123", "issuer", introImportTestPrivateKeyPEM(t))
	if err != nil {
		t.Fatalf("NewClientFromPEM() error: %v", err)
	}

	_, err = fetchSubscriptionPriceDeriveCandidatesByTerritory(context.Background(), client, "target-sub", []string{"SWE"})
	if err == nil || !strings.Contains(err.Error(), `target price point "pp-89" is missing its territory relationship`) {
		t.Fatalf("expected missing territory relationship error, got %v", err)
	}
}

func deriveCandidates(values ...string) []subscriptionPriceDeriveCandidate {
	result := make([]subscriptionPriceDeriveCandidate, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		result = append(result, subscriptionPriceDeriveCandidate{
			PricePointID:  values[i],
			CustomerPrice: values[i+1],
		})
	}
	return result
}
