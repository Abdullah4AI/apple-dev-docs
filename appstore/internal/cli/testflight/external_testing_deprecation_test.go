package testflight

import (
	"strings"
	"testing"
)

func TestFlightDistributionEditExternalTestingDeprecatedHelp(t *testing.T) {
	distribution := TestFlightDistributionCommand()
	var editCommandFound bool
	for _, subcommand := range distribution.Subcommands {
		if subcommand.Name != "edit" {
			continue
		}
		editCommandFound = true

		externalTesting := subcommand.FlagSet.Lookup("external-testing")
		if externalTesting == nil {
			t.Fatal("expected released --external-testing parser surface to remain available")
		}
		if !strings.HasPrefix(externalTesting.Usage, "DEPRECATED:") {
			t.Fatalf("expected deprecated flag help, got %q", externalTesting.Usage)
		}
		if !strings.Contains(subcommand.LongHelp, `asc builds add-groups --build-id "BUILD_ID" --group "GROUP_ID" --submit --confirm`) {
			t.Fatalf("expected enable migration in canonical help, got %q", subcommand.LongHelp)
		}
		if !strings.Contains(subcommand.LongHelp, `asc builds remove-groups --build-id "BUILD_ID" --group "GROUP_ID" --confirm`) {
			t.Fatalf("expected disable migration in canonical help, got %q", subcommand.LongHelp)
		}
	}

	if !editCommandFound {
		t.Fatal("expected testflight distribution edit command")
	}
}
