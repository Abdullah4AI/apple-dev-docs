package cmdtest

import (
	"strings"
	"testing"
)

// TestRemovedRelationshipsAliasCommandsAreUnknown locks that the legacy
// `relationships` command trees, whose deprecated alias factories were deleted
// in 5.0.0, resolve to generic unknown-command usage errors (exit 2) while the
// canonical `links` surfaces remain registered.
func TestRemovedRelationshipsAliasCommandsAreUnknown(t *testing.T) {
	root := RootCommand("1.2.3")

	tests := []struct {
		removed   []string
		canonical []string
	}{
		{removed: []string{"profiles", "relationships"}, canonical: []string{"profiles", "links", "bundle-id"}},
		{removed: []string{"certificates", "relationships"}, canonical: []string{"certificates", "links", "pass-type-id"}},
		{removed: []string{"app-tags", "relationships"}, canonical: []string{"app-tags", "links"}},
		{removed: []string{"app-tags", "territories-relationships"}, canonical: []string{"app-tags", "territories-links"}},
	}

	for _, test := range tests {
		removedPath := strings.Join(test.removed, " ")
		t.Run(removedPath, func(t *testing.T) {
			if findSubcommand(root, test.canonical...) == nil {
				t.Fatalf("canonical command %q not found", strings.Join(test.canonical, " "))
			}
			if findSubcommand(root, test.removed...) != nil {
				t.Fatalf("removed alias command %q is still registered", removedPath)
			}

			args := append(append([]string{}, test.removed...), "--id", "RESOURCE_ID")
			assertUsageExit(t, args, "Error: unknown command `asc "+removedPath+"`")
		})
	}
}
