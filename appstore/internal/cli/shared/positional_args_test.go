package shared

import (
	"errors"
	"flag"
	"testing"
)

func TestRejectPositionalArgs(t *testing.T) {
	if err := RejectPositionalArgs(nil); err != nil {
		t.Fatalf("RejectPositionalArgs(nil) error = %v", err)
	}

	err := RejectPositionalArgs([]string{"stray\x1b[31m", "extra\nline"})
	if err == nil || err.Error() != "unexpected argument(s): stray[31m extra line" {
		t.Fatalf("RejectPositionalArgs() error = %v", err)
	}
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("RejectPositionalArgs() error = %v, want usage error", err)
	}
}
