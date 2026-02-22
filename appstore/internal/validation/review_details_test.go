package validation

import "testing"

func TestReviewDetailsChecks_MissingDetails(t *testing.T) {
	checks := reviewDetailsChecks(nil)
	if !hasCheckID(checks, "review_details.missing") {
		t.Fatalf("expected review_details.missing check, got %v", checks)
	}
}

func TestReviewDetailsChecks_MissingContactFields(t *testing.T) {
	checks := reviewDetailsChecks(&ReviewDetails{
		ID: "detail-1",
		// All contact fields empty.
	})

	needFields := map[string]bool{
		"contactFirstName": false,
		"contactLastName":  false,
		"contactEmail":     false,
		"contactPhone":     false,
	}

	for _, c := range checks {
		if c.ID != "review_details.missing_field" {
			continue
		}
		if _, ok := needFields[c.Field]; ok {
			needFields[c.Field] = true
		}
	}

	for field, found := range needFields {
		if !found {
			t.Fatalf("expected missing field check for %s, got %v", field, checks)
		}
	}
}

func TestReviewDetailsChecks_DemoAccountRequiredMissingCredentials(t *testing.T) {
	checks := reviewDetailsChecks(&ReviewDetails{
		ID:                  "detail-1",
		ContactFirstName:    "A",
		ContactLastName:     "B",
		ContactEmail:        "a@example.com",
		ContactPhone:        "123",
		DemoAccountRequired: true,
		// Missing demo account name/password.
	})

	needFields := map[string]bool{
		"demoAccountName":     false,
		"demoAccountPassword": false,
	}

	for _, c := range checks {
		if c.ID != "review_details.missing_field" {
			continue
		}
		if _, ok := needFields[c.Field]; ok {
			needFields[c.Field] = true
		}
	}

	for field, found := range needFields {
		if !found {
			t.Fatalf("expected missing field check for %s, got %v", field, checks)
		}
	}
}

func TestReviewDetailsChecks_Pass(t *testing.T) {
	checks := reviewDetailsChecks(&ReviewDetails{
		ID:                  "detail-1",
		ContactFirstName:    "A",
		ContactLastName:     "B",
		ContactEmail:        "a@example.com",
		ContactPhone:        "123",
		DemoAccountRequired: false,
	})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestReviewDetailsChecks_PassWithDemoAccount(t *testing.T) {
	checks := reviewDetailsChecks(&ReviewDetails{
		ID:                  "detail-1",
		ContactFirstName:    "A",
		ContactLastName:     "B",
		ContactEmail:        "a@example.com",
		ContactPhone:        "123",
		DemoAccountRequired: true,
		DemoAccountName:     "demo",
		DemoAccountPassword: "pass",
	})
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}
