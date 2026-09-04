package capabilities

import (
	"slices"
	"strings"
	"testing"
)

// The capability is fully public-API backed: read through versions JSON, write
// through asc versions update --downloadable. No web-session command is left in
// the row after the retired asc web apps last-compatible-version view.
func TestLastCompatibleVersionCapabilityUsesPublicAPICommands(t *testing.T) {
	for _, capability := range capabilityRows() {
		if capability.Capability != "Last-compatible version settings" {
			continue
		}
		if capability.Status != statusCLISupported {
			t.Fatalf("status = %q, want %q", capability.Status, statusCLISupported)
		}
		for _, want := range []string{
			"asc versions list --paginate --output json",
			"asc versions view --output json",
			"asc versions update --downloadable",
		} {
			if !slices.Contains(capability.Commands, want) {
				t.Fatalf("missing command %q: %+v", want, capability.Commands)
			}
		}
		for _, command := range capability.Commands {
			if strings.Contains(command, "last-compatible-version") {
				t.Fatalf("retired web-session command still listed: %q", command)
			}
		}
		if strings.Contains(capability.NextAction, "last-compatible-version") {
			t.Fatalf("retired web-session command still in next action: %q", capability.NextAction)
		}
		for _, want := range []string{
			"asc versions update --version-id VERSION_ID --downloadable true",
			"asc versions update --version-id VERSION_ID --downloadable false --confirm",
		} {
			if !strings.Contains(capability.NextAction, want) {
				t.Fatalf("next action missing %q: %q", want, capability.NextAction)
			}
		}
		for _, note := range capability.Notes {
			if strings.Contains(note, "does not currently request or print") {
				t.Fatalf("stale public-client claim remains: %q", note)
			}
			if strings.Contains(note, "last-compatible-version") {
				t.Fatalf("retired web-session command still in notes: %q", note)
			}
		}
		return
	}
	t.Fatal("last-compatible version capability catalog entry not found")
}
