package validate

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestCurrentAppPaidPricingEvidenceUsesLatestActiveManualPrice(t *testing.T) {
	page := &asc.AppPricesResponse{
		Data: []asc.Resource[asc.AppPriceAttributes]{
			{ID: "old", Attributes: asc.AppPriceAttributes{StartDate: "2025-01-01", EndDate: "2025-12-31"}, Relationships: json.RawMessage(`{"appPricePoint":{"data":{"id":"free"}}}`)},
			{ID: "current", Attributes: asc.AppPriceAttributes{StartDate: "2026-01-01"}, Relationships: json.RawMessage(`{"appPricePoint":{"data":{"id":"paid"}}}`)},
		},
		Included: json.RawMessage(`[
			{"type":"appPricePoints","id":"free","attributes":{"customerPrice":"0.00"}},
			{"type":"appPricePoints","id":"paid","attributes":{"customerPrice":"4.99"}}
		]`),
	}

	paid, known := currentAppPaidPricingEvidence([]*asc.AppPricesResponse{page}, time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC))
	if !known || !paid {
		t.Fatalf("pricing evidence = paid %t known %t, want paid and known", paid, known)
	}
}

func TestCurrentAppPaidPricingEvidenceRecognizesCurrentFreePrice(t *testing.T) {
	page := &asc.AppPricesResponse{
		Data: []asc.Resource[asc.AppPriceAttributes]{
			{ID: "current", Attributes: asc.AppPriceAttributes{StartDate: "2026-01-01"}, Relationships: json.RawMessage(`{"appPricePoint":{"data":{"id":"free"}}}`)},
		},
		Included: json.RawMessage(`[
			{"type":"appPricePoints","id":"free","attributes":{"customerPrice":"0.00"}}
		]`),
	}

	paid, known := currentAppPaidPricingEvidence([]*asc.AppPricesResponse{page}, time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC))
	if !known || paid {
		t.Fatalf("pricing evidence = paid %t known %t, want free and known", paid, known)
	}
}

func TestCurrentAppPaidPricingEvidenceFailsClosedOnIncompleteCurrentPrice(t *testing.T) {
	page := &asc.AppPricesResponse{
		Data: []asc.Resource[asc.AppPriceAttributes]{
			{ID: "current", Attributes: asc.AppPriceAttributes{StartDate: "2026-01-01"}, Relationships: json.RawMessage(`{"appPricePoint":{"data":{"id":"missing"}}}`)},
		},
	}

	paid, known := currentAppPaidPricingEvidence([]*asc.AppPricesResponse{page}, time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC))
	if paid || known {
		t.Fatalf("pricing evidence = paid %t known %t, want unknown", paid, known)
	}
}

func TestAppPriceActiveOnExcludesPriceOnEndDate(t *testing.T) {
	active, known := appPriceActiveOn(
		asc.AppPriceAttributes{StartDate: "2026-01-01", EndDate: "2026-08-29"},
		time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC),
	)
	if !known || active {
		t.Fatalf("active = %t known = %t, want inactive and known on end date", active, known)
	}
}
