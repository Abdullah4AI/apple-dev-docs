package cmdtest

import (
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

// TestGameCenterInputValidationReturnsUsageExitCode locks the usage-error
// contract for Game Center flag validation: every pre-request flag check must
// print "Error: <message>" to stderr and exit with code 2, not the generic
// runtime failure code.
//
// The table covers both the per-command checks and the two shared metrics
// helpers (runDetailsMetrics and runMetricsQueue), which format the command
// path into the diagnostic instead of hard-coding it.
func TestGameCenterInputValidationReturnsUsageExitCode(t *testing.T) {
	setupUsageExitCodeEnv(t)

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "achievements list limit above maximum",
			args:    []string{"game-center", "achievements", "list", "--app", "123456789", "--limit", "201"},
			wantErr: "game-center achievements list: --limit must be between 1 and 200",
		},
		{
			name:    "achievements list limit below minimum",
			args:    []string{"game-center", "achievements", "list", "--app", "123456789", "--limit", "-1"},
			wantErr: "game-center achievements list: --limit must be between 1 and 200",
		},
		{
			name:    "achievements list non-App-Store-Connect next",
			args:    []string{"game-center", "achievements", "list", "--next", "http://api.appstoreconnect.apple.com/v1/gameCenterAchievements"},
			wantErr: "game-center achievements list: --next must be an App Store Connect URL",
		},
		{
			name:    "achievements list malformed next",
			args:    []string{"game-center", "achievements", "list", "--next", malformedNextURL},
			wantErr: "game-center achievements list: --next must be a valid URL: " + malformedNextURLParseError,
		},
		{
			name:    "groups list limit above maximum",
			args:    []string{"game-center", "groups", "list", "--limit", "201"},
			wantErr: "game-center groups list: --limit must be between 1 and 200",
		},
		{
			name:    "leaderboard-sets v2 list limit above maximum",
			args:    []string{"game-center", "leaderboard-sets", "v2", "list", "--limit", "201"},
			wantErr: "game-center leaderboard-sets v2 list: --limit must be between 1 and 200",
		},
		{
			name:    "enabled-versions compatible-versions limit above maximum",
			args:    []string{"game-center", "enabled-versions", "compatible-versions", "--limit", "201"},
			wantErr: "game-center enabled-versions compatible-versions: --limit must be between 1 and 200",
		},
		{
			name:    "app-versions compatibility list invalid next",
			args:    []string{"game-center", "app-versions", "compatibility", "list", "--next", "http://api.appstoreconnect.apple.com/v1/x"},
			wantErr: "game-center app-versions compatibility list: --next must be an App Store Connect URL",
		},
		{
			name:    "details metrics limit above maximum",
			args:    []string{"game-center", "details", "metrics", "classic-matchmaking", "--limit", "201"},
			wantErr: "game-center details metrics classic-matchmaking: --limit must be between 1 and 200",
		},
		{
			name:    "details metrics invalid next",
			args:    []string{"game-center", "details", "metrics", "classic-matchmaking", "--next", "http://api.appstoreconnect.apple.com/v1/x"},
			wantErr: "game-center details metrics classic-matchmaking: --next must be an App Store Connect URL",
		},
		{
			name:    "matchmaking metrics limit above maximum",
			args:    []string{"game-center", "matchmaking", "metrics", "queue-sizes", "--limit", "201"},
			wantErr: "game-center matchmaking metrics queue-sizes: --limit must be between 1 and 200",
		},
		{
			name:    "matchmaking metrics invalid next",
			args:    []string{"game-center", "matchmaking", "metrics", "queue-sizes", "--next", "http://api.appstoreconnect.apple.com/v1/x"},
			wantErr: "game-center matchmaking metrics queue-sizes: --next must be an App Store Connect URL",
		},
		{
			name:    "matchmaking queues list limit above maximum",
			args:    []string{"game-center", "matchmaking", "queues", "list", "--limit", "201"},
			wantErr: "game-center matchmaking queues list: --limit must be between 1 and 200",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertUsageExitCode(t, test.args, test.wantErr)
		})
	}
}

// setupUsageExitCodeEnv isolates auth and app-id state so a validation failure
// cannot be masked by a credential lookup or an ambient ASC_APP_ID.
func setupUsageExitCodeEnv(t *testing.T) {
	t.Helper()

	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")
}

// malformedNextURL and malformedNextURLParseError spell out the url.Parse
// diagnostic shared.ValidateNextURL wraps. The message is written out rather
// than recomputed because staticcheck rejects url.Parse on a constant invalid
// URL (SA1007).
const (
	malformedNextURL           = "https://api.appstoreconnect.apple.com/%zz"
	malformedNextURLParseError = `parse "` + malformedNextURL + `": invalid URL escape "%zz"`
)

// assertUsageExitCode runs one invalid invocation and asserts the full usage
// contract: a usage-class error, exit code 2, no stdout, and exactly one
// "Error: <message>" diagnostic on stderr.
func assertUsageExitCode(t *testing.T, args []string, wantErr string) {
	t.Helper()

	stdout, stderr, runErr := runCommand(t, args)

	if runErr == nil {
		t.Fatal("expected error, got nil")
	}
	if got := rootcmd.ExitCodeFromError(runErr); got != rootcmd.ExitUsage {
		t.Fatalf("exit code = %d, want %d (err=%v)", got, rootcmd.ExitUsage, runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	assertUsageErrorStderr(t, trimDeprecationWarnings(stderr), wantErr)
}

// trimDeprecationWarnings drops the leading deprecation warning lines a
// deprecated command or flag alias emits before its diagnostic, so the
// usage-error assertion sees the same stderr shape for deprecated and current
// commands. Only lines matching the documented deprecation form recognized by
// isDeprecatedCommandWarning are removed, so an unrelated warning still
// reaches the assertion. Only the leading block is considered; an unexpected
// warning after the diagnostic still fails the assertion.
func trimDeprecationWarnings(stderr string) string {
	for {
		line, rest, found := strings.Cut(stderr, "\n")
		if !isDeprecatedCommandWarning(strings.TrimSpace(line)) {
			return stderr
		}
		if !found {
			return ""
		}
		stderr = rest
	}
}

// TestTrimDeprecationWarningsKeepsUnrelatedWarnings locks the narrow contract
// of trimDeprecationWarnings: it removes only the documented deprecation
// warning lines, so an unrelated leading warning still reaches the usage-error
// assertion instead of being silently swallowed.
func TestTrimDeprecationWarningsKeepsUnrelatedWarnings(t *testing.T) {
	t.Parallel()

	const diagnostic = "Error: game-center achievements list: --limit must be between 1 and 200\n"

	tests := []struct {
		name   string
		stderr string
		want   string
	}{
		{
			name:   "no warnings",
			stderr: diagnostic,
			want:   diagnostic,
		},
		{
			name:   "deprecated flag alias warning is dropped",
			stderr: "Warning: `--id` is deprecated. Use `--localization-id`.\n" + diagnostic,
			want:   diagnostic,
		},
		{
			name: "deprecated command warning is dropped",
			stderr: "Warning: `asc iap localizations update` is deprecated by App Store Connect API 4.4.1. " +
				"Use `asc iap versions localizations update`.\n" + diagnostic,
			want: diagnostic,
		},
		{
			name:   "unrelated warning is kept",
			stderr: "Warning: something else happened\n" + diagnostic,
			want:   "Warning: something else happened\n" + diagnostic,
		},
		{
			name: "unrelated warning after a deprecation warning is kept",
			stderr: "Warning: `--id` is deprecated. Use `--localization-id`.\n" +
				"Warning: something else happened\n" + diagnostic,
			want: "Warning: something else happened\n" + diagnostic,
		},
		{
			name:   "trailing deprecation warning without a newline",
			stderr: "Warning: `--id` is deprecated. Use `--localization-id`.",
			want:   "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := trimDeprecationWarnings(test.stderr); got != test.want {
				t.Fatalf("trimDeprecationWarnings(%q) = %q, want %q", test.stderr, got, test.want)
			}
		})
	}
}
