package validation

import "testing"

func TestPricingChecks_MissingSchedule(t *testing.T) {
	checks := pricingChecks("app-1", "")
	if !hasCheckID(checks, "pricing.schedule.missing") {
		t.Fatalf("expected pricing.schedule.missing check, got %v", checks)
	}
}

func TestPricingChecks_Pass(t *testing.T) {
	checks := pricingChecks("app-1", "sched-1")
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestAvailabilityChecks_MissingAvailability(t *testing.T) {
	checks := availabilityChecks("app-1", "", 0)
	if !hasCheckID(checks, "availability.missing") {
		t.Fatalf("expected availability.missing check, got %v", checks)
	}
}

func TestAvailabilityChecks_NoTerritories(t *testing.T) {
	checks := availabilityChecks("app-1", "avail-1", 0)
	if !hasCheckID(checks, "availability.territories.none") {
		t.Fatalf("expected availability.territories.none check, got %v", checks)
	}
}

func TestAvailabilityChecks_Pass(t *testing.T) {
	checks := availabilityChecks("app-1", "avail-1", 3)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}
