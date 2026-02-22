package iap

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestParseIAPPriceScheduleIncluded_DecodesDatesFromResourceID(t *testing.T) {
	raw := []byte(`[
		{
			"type":"inAppPurchasePrices",
			"id":"eyJzIjoiNjc0OTI3MzQ1NyIsInQiOiJNVVMiLCJwIjoiMTAzNTciLCJzZCI6MC4wLCJlZCI6MTc3MTIyODgwMC4wMDAwMDAwMDB9",
			"attributes":{}
		},
		{
			"type":"inAppPurchasePrices",
			"id":"eyJzIjoiNjc0OTI3MzQ1NyIsInQiOiJNVVMiLCJwIjoiMTAzODciLCJzZCI6MTc3MTIyODgwMC4wMDAwMDAwMDAsImVkIjowLjB9",
			"attributes":{}
		}
	]`)

	entries, _, err := parseIAPPriceScheduleIncluded(raw)
	if err != nil {
		t.Fatalf("parseIAPPriceScheduleIncluded returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	now := time.Date(2026, time.February, 7, 0, 0, 0, 0, time.UTC)
	changes := buildScheduledChanges(entries, now, "")
	if len(changes) != 1 {
		t.Fatalf("expected 1 scheduled change, got %d", len(changes))
	}

	change := changes[0]
	if change.Territory != "MUS" {
		t.Fatalf("expected territory MUS, got %q", change.Territory)
	}
	if change.FromPricePoint != "10357" {
		t.Fatalf("expected from price point 10357, got %q", change.FromPricePoint)
	}
	if change.ToPricePoint != "10387" {
		t.Fatalf("expected to price point 10387, got %q", change.ToPricePoint)
	}
	if change.EffectiveDate != "2026-02-16" {
		t.Fatalf("expected effective date 2026-02-16, got %q", change.EffectiveDate)
	}
}

func TestResolveIAPPriceSummaries_ContextCancelledReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	summaries, err := resolveIAPPriceSummaries(
		ctx,
		nil,
		[]asc.Resource[asc.InAppPurchaseV2Attributes]{
			{ID: "iap-1"},
		},
		"",
		time.Now().UTC(),
	)
	if err == nil {
		t.Fatalf("expected error for cancelled context")
	}
	if !strings.Contains(err.Error(), "context cancelled") {
		t.Fatalf("expected context cancelled error, got %v", err)
	}
	if summaries != nil {
		t.Fatalf("expected nil summaries on cancelled context, got %#v", summaries)
	}
}

func TestParseManualSchedulePricePointValues_DecodesLegacyPricePointIDs(t *testing.T) {
	raw := []byte(`[
		{
			"type":"inAppPurchasePricePoints",
			"id":"eyJzIjoiMTU1OTI5NDEzOSIsInQiOiJVU0EiLCJwIjoiMyJ9",
			"attributes":{"customerPrice":"2.99","proceeds":"2.54"}
		},
		{
			"type":"territories",
			"id":"USA",
			"attributes":{"currency":"USD"}
		}
	]`)

	values, currency, err := parseManualSchedulePricePointValues(raw, "USA")
	if err != nil {
		t.Fatalf("parseManualSchedulePricePointValues returned error: %v", err)
	}
	if currency != "USD" {
		t.Fatalf("expected currency USD, got %q", currency)
	}
	value, ok := values["3"]
	if !ok {
		t.Fatalf("expected decoded point id 3 in values map")
	}
	if value.CustomerPrice != "2.99" || value.Proceeds != "2.54" {
		t.Fatalf("unexpected value: %#v", value)
	}
}
