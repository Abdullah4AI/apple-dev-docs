package versions

import (
	"strings"
	"testing"
)

func TestVersionsListLatestFlagIsExperimental(t *testing.T) {
	t.Parallel()

	flag := VersionsListCommand().FlagSet.Lookup("latest")
	if flag == nil {
		t.Fatal("expected --latest flag")
	}
	if !strings.HasPrefix(flag.Usage, "[experimental] ") {
		t.Fatalf("--latest usage = %q, want [experimental] prefix", flag.Usage)
	}
}
