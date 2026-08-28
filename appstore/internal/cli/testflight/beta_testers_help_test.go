package testflight

import (
	"strings"
	"testing"
)

func TestBetaTestersListHelpUsesPublicGroupMembershipCommand(t *testing.T) {
	help := BetaTestersListCommand().LongHelp
	want := `asc testflight testers groups list --id "TESTER_ID" --paginate`
	if !strings.Contains(help, want) {
		t.Fatalf("help missing public group-membership command %q: %q", want, help)
	}
	if strings.Contains(help, "asc testflight beta-testers beta-groups") {
		t.Fatalf("help exposes internal group-membership command path: %q", help)
	}
}
