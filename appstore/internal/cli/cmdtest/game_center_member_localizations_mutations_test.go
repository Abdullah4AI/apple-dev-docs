package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestGameCenterMemberLocalizationsCreateValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		stderrSubstr string
	}{
		{
			name:         "missing leaderboard-set-id",
			args:         []string{"game-center", "leaderboard-sets", "member-localizations", "create", "--leaderboard-id", "LB_ID", "--locale", "en-US", "--name", "Test"},
			stderrSubstr: "Error: --leaderboard-set-id is required",
		},
		{
			name:         "missing leaderboard-id",
			args:         []string{"game-center", "leaderboard-sets", "member-localizations", "create", "--leaderboard-set-id", "SET_ID", "--locale", "en-US", "--name", "Test"},
			stderrSubstr: "Error: --leaderboard-id is required",
		},
		{
			name:         "missing locale",
			args:         []string{"game-center", "leaderboard-sets", "member-localizations", "create", "--leaderboard-set-id", "SET_ID", "--leaderboard-id", "LB_ID", "--name", "Test"},
			stderrSubstr: "Error: --locale is required",
		},
		{
			name:         "missing name",
			args:         []string{"game-center", "leaderboard-sets", "member-localizations", "create", "--leaderboard-set-id", "SET_ID", "--leaderboard-id", "LB_ID", "--locale", "en-US"},
			stderrSubstr: "Error: --name is required",
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

func TestGameCenterMemberLocalizationsUpdateValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		stderrSubstr string
	}{
		{
			name:         "missing id",
			args:         []string{"game-center", "leaderboard-sets", "member-localizations", "update", "--name", "New Name"},
			stderrSubstr: "Error: --id is required",
		},
		{
			name:         "no update flags",
			args:         []string{"game-center", "leaderboard-sets", "member-localizations", "update", "--id", "LOCALIZATION_ID"},
			stderrSubstr: "Error: at least one update flag is required (--name)",
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

func TestGameCenterMemberLocalizationsDeleteValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		stderrSubstr string
	}{
		{
			name:         "missing id",
			args:         []string{"game-center", "leaderboard-sets", "member-localizations", "delete", "--confirm"},
			stderrSubstr: "Error: --id is required",
		},
		{
			name:         "missing confirm",
			args:         []string{"game-center", "leaderboard-sets", "member-localizations", "delete", "--id", "LOCALIZATION_ID"},
			stderrSubstr: "Error: --confirm is required",
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
