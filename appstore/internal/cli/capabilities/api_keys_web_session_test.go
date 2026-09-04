package capabilities

import (
	"strings"
	"testing"
)

func TestAPIKeyWebSessionNotesDoNotOverclaimIndividualKeys(t *testing.T) {
	for _, capability := range capabilityRows() {
		if capability.Capability != "App Store Connect API key web-session management" {
			continue
		}
		notes := strings.Join(capability.Notes, " ")
		lower := strings.ToLower(notes)
		if !strings.Contains(lower, "individual") || !strings.Contains(lower, "list") {
			t.Fatalf("expected notes to mention individual-key listing, got %q", notes)
		}
		if !strings.Contains(lower, "view and create-with-p8 are team-key-only") ||
			!strings.Contains(lower, "individual keys appear in list but are not loaded by view") {
			t.Fatalf("notes do not describe API-key operation scope accurately: %q", notes)
		}
		return
	}
	t.Fatal("App Store Connect API key web-session management catalog entry not found")
}
