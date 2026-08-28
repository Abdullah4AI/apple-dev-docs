package validate

import (
	"strings"
	"testing"
)

func TestValidateHelpDocumentsPlaceholderWarningScope(t *testing.T) {
	cmd := ValidateCommand()
	for _, want := range []string{
		"Placeholder copy in localized listing fields",
		"warning; --strict to block",
		"localized TODO copy without marker punctuation",
		"shorter Lorem Ipsum product wording",
	} {
		if !strings.Contains(cmd.LongHelp, want) {
			t.Fatalf("LongHelp missing %q:\n%s", want, cmd.LongHelp)
		}
	}
}
