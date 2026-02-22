package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestGameCenterDetailsCreateValidationErrors(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	tests := []struct {
		name         string
		args         []string
		stderrSubstr string
	}{
		{
			name:         "missing app",
			args:         []string{"game-center", "details", "create"},
			stderrSubstr: "Error: --app is required (or set ASC_APP_ID)",
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

func TestGameCenterDetailsUpdateValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		stderrSubstr string
	}{
		{
			name:         "missing id",
			args:         []string{"game-center", "details", "update", "--challenge-enabled", "true"},
			stderrSubstr: "Error: --id is required",
		},
		{
			name:         "no update flags",
			args:         []string{"game-center", "details", "update", "--id", "DETAIL_ID"},
			stderrSubstr: "Error: at least one update flag is required (--game-center-group-id, --default-leaderboard-id)",
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

func TestGameCenterDetailsCreateInvalidChallengeEnabled(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"game-center", "details", "create", "--app", "123456", "--challenge-enabled", "true"}); err != nil {
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
	if !strings.Contains(stderr, "Error: --challenge-enabled is deprecated and no longer supported by App Store Connect") {
		t.Fatalf("expected stderr to contain deprecated --challenge-enabled error, got %q", stderr)
	}
}

func TestGameCenterDetailsUpdateInvalidChallengeEnabled(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"game-center", "details", "update", "--id", "DETAIL_ID", "--challenge-enabled", "true"}); err != nil {
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
	if !strings.Contains(stderr, "Error: --challenge-enabled is deprecated and no longer supported by App Store Connect") {
		t.Fatalf("expected stderr to contain deprecated --challenge-enabled error, got %q", stderr)
	}
}
