package validation

import "testing"

func TestCategoryChecks_MissingPrimary(t *testing.T) {
	checks := categoryChecks("info-1", "")
	if !hasCheckID(checks, "categories.primary_missing") {
		t.Fatalf("expected categories.primary_missing check, got %v", checks)
	}
}

func TestCategoryChecks_Pass(t *testing.T) {
	checks := categoryChecks("info-1", "cat-1")
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}
