package versions

import (
	"strings"
	"testing"
)

func TestVersionsCustomerReviewsListHelpUsesAPITerritoryCode(t *testing.T) {
	help := VersionsCustomerReviewsListCommand().LongHelp
	if !strings.Contains(help, `--territory USA --sort`) {
		t.Fatalf("expected a three-letter API territory in the example, got %q", help)
	}
	if strings.Contains(help, `--territory US --sort`) {
		t.Fatalf("example must not use an API-invalid two-letter territory, got %q", help)
	}
}
