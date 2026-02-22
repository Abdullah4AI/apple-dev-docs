package validation

import "testing"

func TestIAPReviewReadinessChecks_Empty(t *testing.T) {
	checks := iapReviewReadinessChecks(nil)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestIAPReviewReadinessChecks_WarnsForReadyToSubmit(t *testing.T) {
	checks := iapReviewReadinessChecks([]IAP{
		{ID: "iap-1", Name: "Pro", ProductID: "com.example.pro", State: "READY_TO_SUBMIT"},
	})
	if !hasCheckID(checks, "iap.review_readiness.needs_attention") {
		t.Fatalf("expected warning check, got %v", checks)
	}
	if checks[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity, got %s", checks[0].Severity)
	}
}

func TestIAPReviewReadinessChecks_AllowsInReview(t *testing.T) {
	checks := iapReviewReadinessChecks([]IAP{
		{ID: "iap-1", State: "IN_REVIEW"},
		{ID: "iap-2", State: "WAITING_FOR_REVIEW"},
		{ID: "iap-3", State: "APPROVED"},
	})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestIAPReviewReadinessChecks_IgnoresRemovedFromSale(t *testing.T) {
	checks := iapReviewReadinessChecks([]IAP{
		{ID: "iap-1", State: "REMOVED_FROM_SALE"},
		{ID: "iap-2", State: "DEVELOPER_REMOVED_FROM_SALE"},
	})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}
