package cmdtest

import (
	"strings"
	"testing"
)

// TestSetupHiddenSpellingsRemainAccepted locks the undocumented synonyms on
// `iap setup` and `subscriptions setup`. They were never deprecated, so 5.0.0
// keeps them: each spelling parses, mirrors its canonical flag, and stays out
// of the rendered help until a release announces a deprecation window.
func TestSetupHiddenSpellingsRemainAccepted(t *testing.T) {
	tests := []struct {
		path      []string
		spellings map[string]string
	}{
		{
			path: []string{"iap", "setup"},
			spellings: map[string]string{
				"ref-name": "reference-name",
				"name":     "display-name",
			},
		},
		{
			path: []string{"subscriptions", "setup"},
			spellings: map[string]string{
				"group-ref-name": "group-reference-name",
				"ref-name":       "reference-name",
				"name":           "display-name",
			},
		},
	}

	root := RootCommand("1.2.3")
	for _, test := range tests {
		command := findSubcommand(root, test.path...)
		if command == nil {
			t.Fatalf("command %q not found", strings.Join(test.path, " "))
		}
		usage := command.UsageFunc(command)
		for spelling, canonical := range test.spellings {
			name := strings.Join(test.path, " ") + " --" + spelling
			t.Run(name, func(t *testing.T) {
				if command.FlagSet.Lookup(canonical) == nil {
					t.Fatalf("canonical flag --%s not registered", canonical)
				}
				if command.FlagSet.Lookup(spelling) == nil {
					t.Fatalf("hidden spelling --%s must stay registered", spelling)
				}
				if strings.Contains(usage, "\n  --"+spelling+" ") {
					t.Fatalf("hidden spelling --%s must stay out of help: %q", spelling, usage)
				}
			})
		}
	}
}
