package validation

import "testing"

func TestSubscriptionReviewReadinessChecks_Empty(t *testing.T) {
	checks := subscriptionReviewReadinessChecks(nil)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestSubscriptionReviewReadinessChecks_WarnsForReadyToSubmit(t *testing.T) {
	checks := subscriptionReviewReadinessChecks([]Subscription{
		{ID: "sub-1", Name: "Monthly", ProductID: "com.example.monthly", State: "READY_TO_SUBMIT"},
	})
	if !hasCheckID(checks, "subscriptions.review_readiness.needs_attention") {
		t.Fatalf("expected warning check, got %v", checks)
	}
	if checks[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity, got %s", checks[0].Severity)
	}
}

func TestSubscriptionReviewReadinessChecks_AllowsApproved(t *testing.T) {
	checks := subscriptionReviewReadinessChecks([]Subscription{
		{ID: "sub-1", State: "APPROVED"},
		{ID: "sub-2", State: "IN_REVIEW"},
		{ID: "sub-3", State: "WAITING_FOR_REVIEW"},
	})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestSubscriptionReviewReadinessChecks_IgnoresRemovedFromSale(t *testing.T) {
	checks := subscriptionReviewReadinessChecks([]Subscription{
		{ID: "sub-1", State: "REMOVED_FROM_SALE"},
		{ID: "sub-2", State: "DEVELOPER_REMOVED_FROM_SALE"},
	})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestSubscriptionImageChecks_WarnsWhenImageMissing(t *testing.T) {
	checks := subscriptionImageChecks([]Subscription{
		{ID: "sub-1", Name: "Monthly", ProductID: "com.example.monthly"},
	})
	if !hasCheckID(checks, "subscriptions.images.recommended") {
		t.Fatalf("expected image check, got %v", checks)
	}
	if checks[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity, got %s", checks[0].Severity)
	}
	if checks[0].Remediation == "" {
		t.Fatalf("expected remediation explaining why image matters, got %+v", checks[0])
	}
}

func TestSubscriptionFetchChecks_AddsInfoWhenSkipped(t *testing.T) {
	checks := subscriptionFetchChecks("subscription permissions unavailable")
	if !hasCheckID(checks, "subscriptions.readiness.unverified") {
		t.Fatalf("expected readiness skip check, got %v", checks)
	}
	if checks[0].Severity != SeverityInfo {
		t.Fatalf("expected info severity, got %s", checks[0].Severity)
	}
}

func TestSubscriptionImageChecks_AllowsSubscriptionsWithImages(t *testing.T) {
	checks := subscriptionImageChecks([]Subscription{
		{ID: "sub-1", HasImage: true},
	})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestSubscriptionImageChecks_IgnoresRemovedFromSale(t *testing.T) {
	checks := subscriptionImageChecks([]Subscription{
		{ID: "sub-1", State: "REMOVED_FROM_SALE"},
		{ID: "sub-2", State: "DEVELOPER_REMOVED_FROM_SALE"},
	})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestSubscriptionImageChecks_AddsInfoWhenImageCheckSkipped(t *testing.T) {
	checks := subscriptionImageChecks([]Subscription{
		{
			ID:                   "sub-1",
			Name:                 "Monthly",
			ProductID:            "com.example.monthly",
			ImageCheckSkipped:    true,
			ImageCheckSkipReason: "permission denied",
		},
	})
	if !hasCheckID(checks, "subscriptions.images.unverified") {
		t.Fatalf("expected unverified image check, got %v", checks)
	}
	if checks[0].Severity != SeverityInfo {
		t.Fatalf("expected info severity, got %s", checks[0].Severity)
	}
}
