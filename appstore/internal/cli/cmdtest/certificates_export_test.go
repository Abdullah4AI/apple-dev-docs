package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestCertificatesExport_MissingRequiredFlags(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"certificates", "export"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected flag.ErrHelp, got %v", err)
		}
		if got, want := err.Error(), "--certificate"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --certificate is required") {
		t.Fatalf("expected missing certificate error, got %q", stderr)
	}
}
