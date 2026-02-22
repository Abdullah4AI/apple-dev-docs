package iap

import (
	"context"
	"testing"
	"time"
)

func TestContextWithAssetUploadTimeout(t *testing.T) {
	ctx, cancel := contextWithAssetUploadTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context deadline to be set")
	}
	if time.Until(deadline) <= 0 {
		t.Fatal("expected future deadline")
	}
}

func TestParseOfferCodeEligibilities(t *testing.T) {
	got, err := parseOfferCodeEligibilities("non_spender,ACTIVE_SPENDER,non_spender")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected deduplicated eligibilities, got %v", got)
	}
	if got[0] != "NON_SPENDER" || got[1] != "ACTIVE_SPENDER" {
		t.Fatalf("unexpected eligibility order/content: %v", got)
	}

	if _, err := parseOfferCodeEligibilities("INVALID"); err == nil {
		t.Fatal("expected validation error for invalid eligibility")
	}
}

func TestParseOfferCodePrices(t *testing.T) {
	prices, err := parseOfferCodePrices("usa:pp-1,jpn:pp-2")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}
	if prices[0].TerritoryID != "USA" || prices[0].PricePointID != "pp-1" {
		t.Fatalf("unexpected first price: %+v", prices[0])
	}

	if _, err := parseOfferCodePrices("usa-pp-1"); err == nil {
		t.Fatal("expected parse error for malformed prices")
	}
}

func TestParsePriceSchedulePrices(t *testing.T) {
	prices, err := parsePriceSchedulePrices("pp1:2026-01-01:2026-02-01,pp2::")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(prices) != 2 {
		t.Fatalf("expected 2 schedule entries, got %d", len(prices))
	}
	if prices[0].PricePointID != "pp1" || prices[0].StartDate != "2026-01-01" || prices[0].EndDate != "2026-02-01" {
		t.Fatalf("unexpected first schedule entry: %+v", prices[0])
	}
	if prices[1].PricePointID != "pp2" || prices[1].StartDate != "" || prices[1].EndDate != "" {
		t.Fatalf("unexpected second schedule entry: %+v", prices[1])
	}

	if _, err := parsePriceSchedulePrices(":2026-01-01"); err == nil {
		t.Fatal("expected parse error for missing price point id")
	}
	if _, err := parsePriceSchedulePrices("pp1:01-01-2026"); err == nil {
		t.Fatal("expected parse error for invalid date format")
	}
}

func TestNormalizeIAPDate(t *testing.T) {
	got, err := normalizeIAPDate("2026-02-10", "--date")
	if err != nil {
		t.Fatalf("unexpected date parse error: %v", err)
	}
	if got != "2026-02-10" {
		t.Fatalf("expected normalized date 2026-02-10, got %q", got)
	}

	if _, err := normalizeIAPDate("10-02-2026", "--date"); err == nil {
		t.Fatal("expected date format validation error")
	}
}
