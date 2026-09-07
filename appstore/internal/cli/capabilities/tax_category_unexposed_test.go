package capabilities

import (
	"slices"
	"strings"
	"testing"
)

const taxCategoryCapabilityName = "App and In-App Purchase tax category"

// Both tax-category scopes are available through their web-session commands.
func TestTaxCategoryIsReportedAsWebSession(t *testing.T) {
	for _, capability := range capabilityRows() {
		if capability.Capability != taxCategoryCapabilityName {
			continue
		}
		if capability.Status != statusWebSession {
			t.Fatalf("expected %q status %q, got %q", taxCategoryCapabilityName, statusWebSession, capability.Status)
		}
		if capability.Area != "monetization" {
			t.Fatalf("expected %q area %q, got %q", taxCategoryCapabilityName, "monetization", capability.Area)
		}
		wantCommands := []string{
			"asc web apps tax-category list",
			"asc web apps tax-category view",
			"asc web apps tax-category set",
			"asc web iap tax-category list",
			"asc web iap tax-category view",
			"asc web iap tax-category set",
			"asc web iap tax-category reset",
		}
		if !slices.Equal(capability.Commands, wantCommands) {
			t.Fatalf("expected %q commands %v, got %v", taxCategoryCapabilityName, wantCommands, capability.Commands)
		}
		if strings.TrimSpace(capability.NextAction) == "" {
			t.Fatalf("expected %q to carry a next action", taxCategoryCapabilityName)
		}
		if len(capability.Notes) == 0 {
			t.Fatalf("expected %q to carry explanatory notes", taxCategoryCapabilityName)
		}
		return
	}

	t.Fatalf("capability %q not found", taxCategoryCapabilityName)
}

func TestTaxCategoryCapabilityDistinguishesCatalogs(t *testing.T) {
	for _, capability := range capabilityRows() {
		if capability.Capability != taxCategoryCapabilityName {
			continue
		}
		joined := strings.ToLower(strings.Join(capability.Notes, " "))
		if !strings.Contains(joined, "addon") || !strings.Contains(joined, "application") {
			t.Fatalf("expected %q notes to distinguish the ADDON and APPLICATION catalogs, got %v", taxCategoryCapabilityName, capability.Notes)
		}
		return
	}
	t.Fatalf("capability %q not found", taxCategoryCapabilityName)
}
