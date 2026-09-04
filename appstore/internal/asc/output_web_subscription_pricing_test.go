package asc

import "testing"

func TestWebSubscriptionMonthlyCommitmentBootstrapRows(t *testing.T) {
	headers, rows := webSubscriptionMonthlyCommitmentBootstrapRows(&WebSubscriptionMonthlyCommitmentBootstrapResult{
		SubscriptionID:      "sub-1",
		Territory:           "NOR",
		PlanAvailabilityID:  "plan-monthly",
		PlanAvailabilityNew: true,
		PricesCreated:       true,
		Verified:            true,
		CompletedStage:      WebMonthlyCommitmentStageVerified,
	})
	if len(headers) != 8 || len(rows) != 1 {
		t.Fatalf("headers=%d rows=%d", len(headers), len(rows))
	}
	if rows[0][0] != "sub-1" || rows[0][5] != "true" || rows[0][6] != WebMonthlyCommitmentStageVerified {
		t.Fatalf("unexpected row: %#v", rows[0])
	}
}

func TestWebSubscriptionMonthlyCommitmentBootstrapRowsIncludeFailure(t *testing.T) {
	_, rows := webSubscriptionMonthlyCommitmentBootstrapRows(&WebSubscriptionMonthlyCommitmentBootstrapResult{
		CompletedStage: WebMonthlyCommitmentStagePrices,
		Failure:        "UPFRONT price record for NOR did not match price point upfront-point",
	})
	if rows[0][7] != "UPFRONT price record for NOR did not match price point upfront-point" {
		t.Fatalf("failure column = %q", rows[0][7])
	}
}

func TestWebSubscriptionMonthlyCommitmentBootstrapRowsDryRun(t *testing.T) {
	headers, rows := webSubscriptionMonthlyCommitmentBootstrapRows(&WebSubscriptionMonthlyCommitmentBootstrapResult{
		SubscriptionID:              "sub-1",
		Territory:                   "NOR",
		PlanAvailabilityWouldCreate: true,
		UpfrontPricePointID:         "upfront-point",
		MonthlyPricePointID:         "monthly-point",
		DryRun:                      true,
		StartDate:                   "2026-10-01",
		PreserveCurrentPrice:        true,
	})
	if len(headers) != 9 || headers[0] != "Dry Run" || headers[8] != "Preserve Current Price" {
		t.Fatalf("unexpected dry-run headers: %#v", headers)
	}
	if rows[0][0] != "true" || rows[0][4] != "true" || rows[0][5] != "upfront-point" || rows[0][7] != "2026-10-01" || rows[0][8] != "true" {
		t.Fatalf("unexpected dry-run row: %#v", rows[0])
	}
}
