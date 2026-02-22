package account

import "testing"

func TestSummarizeAccountChecks(t *testing.T) {
	red := summarizeAccountChecks([]accountCheck{
		{Name: "authentication", Status: "fail", Message: "auth broken"},
		{Name: "api_access", Status: "ok", Message: "ok"},
	})
	if red.Health != "red" {
		t.Fatalf("expected red health, got %q", red.Health)
	}
	if red.ErrorCount != 1 {
		t.Fatalf("expected one error, got %d", red.ErrorCount)
	}
	if red.NextAction != "auth broken" {
		t.Fatalf("unexpected next action %q", red.NextAction)
	}

	yellow := summarizeAccountChecks([]accountCheck{
		{Name: "authentication", Status: "ok", Message: "ok"},
		{Name: "agreements", Status: "unavailable", Message: "not available"},
	})
	if yellow.Health != "yellow" {
		t.Fatalf("expected yellow health, got %q", yellow.Health)
	}
	if yellow.WarningCount != 1 {
		t.Fatalf("expected one warning, got %d", yellow.WarningCount)
	}

	green := summarizeAccountChecks([]accountCheck{
		{Name: "authentication", Status: "ok", Message: "ok"},
		{Name: "api_access", Status: "ok", Message: "ok"},
	})
	if green.Health != "green" {
		t.Fatalf("expected green health, got %q", green.Health)
	}
	if green.NextAction != "No action needed." {
		t.Fatalf("unexpected next action %q", green.NextAction)
	}
}
