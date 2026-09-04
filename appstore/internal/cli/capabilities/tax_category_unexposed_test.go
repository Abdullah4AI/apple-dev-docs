package capabilities

import (
	"strings"
	"testing"
)

const taxCategoryCapabilityName = "App and In-App Purchase tax category"

// Apple exposes tax category only in the App Store Connect web UI. The capability
// registry must say so instead of leaving callers to discover the gap by failing
// against the public API.
func TestTaxCategoryIsReportedAsNotPublicAPI(t *testing.T) {
	for _, capability := range capabilityRows() {
		if capability.Capability != taxCategoryCapabilityName {
			continue
		}
		if capability.Status != statusNotPublicAPI {
			t.Fatalf("expected %q status %q, got %q", taxCategoryCapabilityName, statusNotPublicAPI, capability.Status)
		}
		if capability.Area != "monetization" {
			t.Fatalf("expected %q area %q, got %q", taxCategoryCapabilityName, "monetization", capability.Area)
		}
		if len(capability.Commands) != 0 {
			t.Fatalf("expected %q to advertise no CLI commands, got %v", taxCategoryCapabilityName, capability.Commands)
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

// No registry row may claim a shipped tax-category command while the web-session
// endpoint contract is still uncaptured.
func TestNoCapabilityClaimsATaxCategoryCommand(t *testing.T) {
	for _, capability := range capabilityRows() {
		for _, command := range capability.Commands {
			if strings.Contains(strings.ToLower(command), "tax") {
				t.Fatalf("unexpected tax command %q advertised by %q", command, capability.Capability)
			}
		}
	}
}
