package gamecenter

import (
	"context"
	"errors"
	"flag"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestGameCenterLeaderboardSetMembersReplacementWarnsWithoutConfirmDuringCompatibilityWindow(t *testing.T) {
	isolateGameCenterAuthEnv(t)

	commands := map[string]func() *ffcli.Command{
		"v1": GameCenterLeaderboardSetMembersSetCommand,
		"v2": GameCenterLeaderboardSetMembersV2SetCommand,
	}

	for name, newCommand := range commands {
		for _, leaderboardIDs := range []string{"leaderboard-1", ""} {
			t.Run(name+" ids="+leaderboardIDs, func(t *testing.T) {
				cmd := newCommand()
				if err := cmd.FlagSet.Parse([]string{
					"--set-id", "set-1", "--leaderboard-ids", leaderboardIDs,
				}); err != nil {
					t.Fatalf("parse flags: %v", err)
				}

				var err error
				stderr := captureGameCenterStderr(t, func() {
					err = cmd.Exec(context.Background(), []string{})
				})
				if errors.Is(err, flag.ErrHelp) {
					t.Fatalf("replacement without --confirm must remain compatible before 5.0.0, got %v", err)
				}
				want := gameCenterReplacementConfirmWarning + "\n"
				if stderr != want {
					t.Fatalf("stderr = %q, want %q", stderr, want)
				}
			})
		}
	}
}

func TestGameCenterLeaderboardSetMembersRequiresLeaderboardIDs(t *testing.T) {
	isolateGameCenterAuthEnv(t)

	commands := map[string]func() *ffcli.Command{
		"v1": GameCenterLeaderboardSetMembersSetCommand,
		"v2": GameCenterLeaderboardSetMembersV2SetCommand,
	}

	for name, newCommand := range commands {
		t.Run(name, func(t *testing.T) {
			cmd := newCommand()
			if err := cmd.FlagSet.Parse([]string{
				"--set-id", "set-1", "--confirm",
			}); err != nil {
				t.Fatalf("parse flags: %v", err)
			}

			var err error
			stderr := captureGameCenterStderr(t, func() {
				err = cmd.Exec(context.Background(), []string{})
			})
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("missing --leaderboard-ids should fail validation before auth, got %v", err)
			}
			if err.Error() != "--leaderboard-ids" {
				t.Fatalf("error = %q, want missing parameter %q", err.Error(), "--leaderboard-ids")
			}
			if stderr != "Error: --leaderboard-ids is required\n" {
				t.Fatalf("stderr = %q, want exact missing-ID diagnostic", stderr)
			}
		})
	}
}

func TestGameCenterLeaderboardSetMembersConfirmPassesValidation(t *testing.T) {
	isolateGameCenterAuthEnv(t)

	commands := map[string]func() *ffcli.Command{
		"v1": GameCenterLeaderboardSetMembersSetCommand,
		"v2": GameCenterLeaderboardSetMembersV2SetCommand,
	}

	for name, newCommand := range commands {
		for _, leaderboardIDs := range []string{"leaderboard-1", ""} {
			t.Run(name+" ids="+leaderboardIDs, func(t *testing.T) {
				cmd := newCommand()
				if err := cmd.FlagSet.Parse([]string{
					"--set-id", "set-1", "--leaderboard-ids", leaderboardIDs, "--confirm",
				}); err != nil {
					t.Fatalf("parse flags: %v", err)
				}

				err := cmd.Exec(context.Background(), []string{})
				if errors.Is(err, flag.ErrHelp) {
					t.Fatalf("replacement with --confirm should pass validation before auth, got %v", err)
				}
			})
		}
	}
}
