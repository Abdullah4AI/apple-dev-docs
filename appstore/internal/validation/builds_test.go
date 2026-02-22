package validation

import "testing"

func TestBuildChecks_MissingBuild(t *testing.T) {
	checks := buildChecks(nil)
	if !hasCheckID(checks, "build.required.missing") {
		t.Fatalf("expected build.required.missing check, got %v", checks)
	}
}

func TestBuildChecks_InvalidProcessingState(t *testing.T) {
	checks := buildChecks(&Build{
		ID:              "build-1",
		ProcessingState: "PROCESSING",
	})
	if !hasCheckID(checks, "build.invalid.processing_state") {
		t.Fatalf("expected build.invalid.processing_state check, got %v", checks)
	}
}

func TestBuildChecks_ExpiredBuild(t *testing.T) {
	checks := buildChecks(&Build{
		ID:              "build-1",
		ProcessingState: "VALID",
		Expired:         true,
	})
	if !hasCheckID(checks, "build.invalid.expired") {
		t.Fatalf("expected build.invalid.expired check, got %v", checks)
	}
}

func TestBuildChecks_Pass(t *testing.T) {
	checks := buildChecks(&Build{
		ID:              "build-1",
		ProcessingState: "VALID",
		Expired:         false,
	})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}
