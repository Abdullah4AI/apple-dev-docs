package validation

import "testing"

func TestRemediationStepsOrdersBlockingErrorsBeforeWarningsAndInfos(t *testing.T) {
	checks := []CheckResult{
		{
			ID:          "warning.first",
			Severity:    SeverityWarning,
			Message:     "warning",
			Remediation: "fix warning",
		},
		{
			ID:          "error.second",
			Severity:    SeverityError,
			Message:     "error",
			Remediation: "fix error",
		},
		{
			ID:          "info.third",
			Severity:    SeverityInfo,
			Message:     "info",
			Remediation: "review info",
		},
	}

	steps := RemediationSteps(checks, false)
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	if steps[0].CheckID != "error.second" {
		t.Fatalf("expected error first, got %+v", steps)
	}
	if steps[1].CheckID != "warning.first" {
		t.Fatalf("expected warning second, got %+v", steps)
	}
	if steps[2].CheckID != "info.third" {
		t.Fatalf("expected info third, got %+v", steps)
	}
	if !steps[0].Blocking {
		t.Fatalf("expected error step to be blocking, got %+v", steps[0])
	}
	if steps[1].Blocking {
		t.Fatalf("expected warning step to be non-blocking without strict mode, got %+v", steps[1])
	}
}

func TestBuildRemediationIncludesOrderedPlan(t *testing.T) {
	remediation := BuildRemediation([]CheckResult{
		{
			ID:          "metadata.required.description",
			Severity:    SeverityError,
			Message:     "description is required",
			Remediation: "Provide a description",
		},
		{
			ID:          "metadata.required.whats_new",
			Severity:    SeverityWarning,
			Message:     "what's new is empty",
			Remediation: "Provide release notes",
		},
	}, false)

	if remediation.TotalActionable != 2 {
		t.Fatalf("expected total actionable 2, got %d", remediation.TotalActionable)
	}
	if len(remediation.Steps) != 2 {
		t.Fatalf("expected two ordered remediation steps, got %d", len(remediation.Steps))
	}
	if remediation.Steps[0].CheckID != "metadata.required.description" {
		t.Fatalf("expected first step to be description remediation, got %+v", remediation.Steps[0])
	}
	if remediation.Steps[1].CheckID != "metadata.required.whats_new" {
		t.Fatalf("expected second step to be what's new remediation, got %+v", remediation.Steps[1])
	}
}
