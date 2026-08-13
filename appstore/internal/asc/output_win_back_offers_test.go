package asc

import (
	"encoding/json"
	"testing"
)

func TestWinBackOfferPricesRowsHandlesFreeTrialEntriesWithoutPricePoint(t *testing.T) {
	resp := &WinBackOfferPricesResponse{
		Data: []Resource[WinBackOfferPriceAttributes]{
			{
				Type:          ResourceTypeWinBackOfferPrices,
				ID:            "price-free",
				Relationships: json.RawMessage(`{"territory":{"data":{"type":"territories","id":"USA"}}}`),
			},
			{
				Type:          ResourceTypeWinBackOfferPrices,
				ID:            "price-paid",
				Relationships: json.RawMessage(`{"territory":{"data":{"type":"territories","id":"FRA"}},"subscriptionPricePoint":{"data":{"type":"subscriptionPricePoints","id":"price-point-1"}}}`),
			},
		},
	}

	headers, rows, err := winBackOfferPricesRows(resp)
	if err != nil {
		t.Fatalf("winBackOfferPricesRows() error: %v", err)
	}
	if len(headers) != 3 {
		t.Fatalf("headers = %v, want 3 columns", headers)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %v, want 2 rows", rows)
	}
	if rows[0][0] != "price-free" || rows[0][1] != "USA" || rows[0][2] != "" {
		t.Fatalf("free-trial row = %v, want empty price point", rows[0])
	}
	if rows[1][0] != "price-paid" || rows[1][1] != "FRA" || rows[1][2] != "price-point-1" {
		t.Fatalf("paid row = %v", rows[1])
	}
}
