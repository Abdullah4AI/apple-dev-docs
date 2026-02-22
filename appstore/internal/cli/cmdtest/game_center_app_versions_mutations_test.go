package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestGameCenterAppVersionsCreateValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		stderrSubstr string
	}{
		{
			name:         "missing app-store-version-id",
			args:         []string{"game-center", "app-versions", "create"},
			stderrSubstr: "Error: --app-store-version-id is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected ErrHelp, got %v", err)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.stderrSubstr) {
				t.Fatalf("expected stderr to contain %q, got %q", test.stderrSubstr, stderr)
			}
		})
	}
}

func TestGameCenterAppVersionsUpdateValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		stderrSubstr string
	}{
		{
			name:         "missing id",
			args:         []string{"game-center", "app-versions", "update", "--enabled", "true"},
			stderrSubstr: "Error: --id is required",
		},
		{
			name:         "no update flags",
			args:         []string{"game-center", "app-versions", "update", "--id", "GC_APP_VERSION_ID"},
			stderrSubstr: "Error: at least one update flag is required (--enabled)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected ErrHelp, got %v", err)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.stderrSubstr) {
				t.Fatalf("expected stderr to contain %q, got %q", test.stderrSubstr, stderr)
			}
		})
	}
}

func TestGameCenterAppVersionsUpdateInvalidEnabled(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"game-center", "app-versions", "update", "--id", "GC_APP_VERSION_ID", "--enabled", "maybe"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --enabled must be 'true' or 'false'") {
		t.Fatalf("expected stderr to contain invalid --enabled error, got %q", stderr)
	}
}
