package cmdtest

import (
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

// TestSubscriptionsInputValidationReturnsUsageExitCode locks the usage-error contract
// for subscriptions pre-request flag validation: every check must print
// "Error: <message>" to stderr, naming the command path the operator invoked,
// and exit with code 2 rather than the generic runtime failure code.
//
// Part of #518.
func TestSubscriptionsInputValidationReturnsUsageExitCode(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	tests := []struct {
		name    string
		args    []string
		wantErr string
		// deprecated marks a command that prints a deprecation notice ahead
		// of its diagnostic, so the diagnostic is not the first stderr line.
		deprecated bool
	}{
		{
			name:    "groups list app 123 limit 201",
			args:    []string{"subscriptions", "groups", "list", "--app", "123", "--limit", "201"},
			wantErr: "subscriptions groups list: --limit must be between 1 and 200",
		},
		{
			name:    "groups list app 123 next non apple url",
			args:    []string{"subscriptions", "groups", "list", "--app", "123", "--next", "http://example.com/x"},
			wantErr: "subscriptions groups list: --next must be an App Store Connect URL",
		},
		{
			name:       "groups localizations list group-id g limit 201",
			args:       []string{"subscriptions", "groups", "localizations", "list", "--group-id", "g", "--limit", "201"},
			wantErr:    "subscriptions groups localizations list: --limit must be between 1 and 200",
			deprecated: true,
		},
		{
			name:       "groups localizations list group-id g next non apple url",
			args:       []string{"subscriptions", "groups", "localizations", "list", "--group-id", "g", "--next", "http://example.com/x"},
			wantErr:    "subscriptions groups localizations list: --next must be an App Store Connect URL",
			deprecated: true,
		},
		{
			name:    "list group-id g limit 201",
			args:    []string{"subscriptions", "list", "--group-id", "g", "--limit", "201"},
			wantErr: "subscriptions list: --limit must be between 1 and 200",
		},
		{
			name:    "list group-id g next non apple url",
			args:    []string{"subscriptions", "list", "--group-id", "g", "--next", "http://example.com/x"},
			wantErr: "subscriptions list: --next must be an App Store Connect URL",
		},
		{
			name:    "pricing prices list subscription-id s limit 201",
			args:    []string{"subscriptions", "pricing", "prices", "list", "--subscription-id", "s", "--limit", "201"},
			wantErr: "subscriptions pricing prices list: --limit must be between 1 and 200",
		},
		{
			name:    "pricing prices list subscription-id s next non apple url",
			args:    []string{"subscriptions", "pricing", "prices", "list", "--subscription-id", "s", "--next", "http://example.com/x"},
			wantErr: "subscriptions pricing prices list: --next must be an App Store Connect URL",
		},
		{
			name:    "pricing price-points list subscription-id s limit 201",
			args:    []string{"subscriptions", "pricing", "price-points", "list", "--subscription-id", "s", "--limit", "201"},
			wantErr: "subscriptions pricing price-points list: --limit must be between 1 and 200",
		},
		{
			name:    "pricing price-points list subscription-id s next non apple url",
			args:    []string{"subscriptions", "pricing", "price-points", "list", "--subscription-id", "s", "--next", "http://example.com/x"},
			wantErr: "subscriptions pricing price-points list: --next must be an App Store Connect URL",
		},
		{
			name:    "offers introductory list subscription-id s limit 201",
			args:    []string{"subscriptions", "offers", "introductory", "list", "--subscription-id", "s", "--limit", "201"},
			wantErr: "subscriptions offers introductory list: --limit must be between 1 and 200",
		},
		{
			name:    "offers introductory list subscription-id s next non apple url",
			args:    []string{"subscriptions", "offers", "introductory", "list", "--subscription-id", "s", "--next", "http://example.com/x"},
			wantErr: "subscriptions offers introductory list: --next must be an App Store Connect URL",
		},
		{
			name:    "offers promotional list subscription-id s limit 201",
			args:    []string{"subscriptions", "offers", "promotional", "list", "--subscription-id", "s", "--limit", "201"},
			wantErr: "subscriptions offers promotional list: --limit must be between 1 and 200",
		},
		{
			name:    "offers promotional list subscription-id s next non apple url",
			args:    []string{"subscriptions", "offers", "promotional", "list", "--subscription-id", "s", "--next", "http://example.com/x"},
			wantErr: "subscriptions offers promotional list: --next must be an App Store Connect URL",
		},
		{
			name:    "offers promotional prices id o limit 201",
			args:    []string{"subscriptions", "offers", "promotional", "prices", "--id", "o", "--limit", "201"},
			wantErr: "subscriptions offers promotional prices: --limit must be between 1 and 200",
		},
		{
			name:    "offers promotional prices id o next non apple url",
			args:    []string{"subscriptions", "offers", "promotional", "prices", "--id", "o", "--next", "http://example.com/x"},
			wantErr: "subscriptions offers promotional prices: --next must be an App Store Connect URL",
		},
		{
			name:    "offers offer-codes list subscription-id s limit 201",
			args:    []string{"subscriptions", "offers", "offer-codes", "list", "--subscription-id", "s", "--limit", "201"},
			wantErr: "subscriptions offers offer-codes list: --limit must be between 1 and 200",
		},
		{
			name:    "offers offer-codes list subscription-id s next non apple url",
			args:    []string{"subscriptions", "offers", "offer-codes", "list", "--subscription-id", "s", "--next", "http://example.com/x"},
			wantErr: "subscriptions offers offer-codes list: --next must be an App Store Connect URL",
		},
		{
			name:    "offers offer-codes one-time-codes list offer-code-id o limit 201",
			args:    []string{"subscriptions", "offers", "offer-codes", "one-time-codes", "list", "--offer-code-id", "o", "--limit", "201"},
			wantErr: "subscriptions offers offer-codes one-time-codes list: --limit must be between 1 and 200",
		},
		{
			name:    "offers offer-codes one-time-codes list offer-code-id o next non apple url",
			args:    []string{"subscriptions", "offers", "offer-codes", "one-time-codes", "list", "--offer-code-id", "o", "--next", "http://example.com/x"},
			wantErr: "subscriptions offers offer-codes one-time-codes list: --next must be an App Store Connect URL",
		},
		{
			name:    "offers offer-codes prices offer-code-id o limit 201",
			args:    []string{"subscriptions", "offers", "offer-codes", "prices", "--offer-code-id", "o", "--limit", "201"},
			wantErr: "subscriptions offers offer-codes prices: --limit must be between 1 and 200",
		},
		{
			name:    "offers offer-codes prices offer-code-id o next non apple url",
			args:    []string{"subscriptions", "offers", "offer-codes", "prices", "--offer-code-id", "o", "--next", "http://example.com/x"},
			wantErr: "subscriptions offers offer-codes prices: --next must be an App Store Connect URL",
		},
		{
			name:       "localizations list subscription-id s limit 201",
			args:       []string{"subscriptions", "localizations", "list", "--subscription-id", "s", "--limit", "201"},
			wantErr:    "subscriptions localizations list: --limit must be between 1 and 200",
			deprecated: true,
		},
		{
			name:       "localizations list subscription-id s next non apple url",
			args:       []string{"subscriptions", "localizations", "list", "--subscription-id", "s", "--next", "http://example.com/x"},
			wantErr:    "subscriptions localizations list: --next must be an App Store Connect URL",
			deprecated: true,
		},
		{
			name:       "images list subscription-id s limit 201",
			args:       []string{"subscriptions", "images", "list", "--subscription-id", "s", "--limit", "201"},
			wantErr:    "subscriptions images list: --limit must be between 1 and 200",
			deprecated: true,
		},
		{
			name:       "images list subscription-id s next non apple url",
			args:       []string{"subscriptions", "images", "list", "--subscription-id", "s", "--next", "http://example.com/x"},
			wantErr:    "subscriptions images list: --next must be an App Store Connect URL",
			deprecated: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, runErr := runCommand(t, test.args)

			if runErr == nil {
				t.Fatal("expected error, got nil")
			}
			if got := rootcmd.ExitCodeFromError(runErr); got != rootcmd.ExitUsage {
				t.Fatalf("exit code = %d, want %d (err=%v)", got, rootcmd.ExitUsage, runErr)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if test.deprecated {
				assertUsageErrorStderrAfterNotices(t, stderr, test.wantErr)
				return
			}
			assertUsageErrorStderr(t, stderr, test.wantErr)
		})
	}
}

// assertUsageErrorStderrAfterNotices is assertUsageErrorStderr for a command
// that prints a deprecation notice ahead of its diagnostic. The diagnostic is
// still the complete, single "Error: <message>" line, but it is not the first
// line of stderr.
func assertUsageErrorStderrAfterNotices(t *testing.T, stderr, wantMessage string) {
	t.Helper()

	wantLine := "Error: " + wantMessage + "\n"
	if !strings.Contains(stderr, wantLine) {
		t.Fatalf("stderr = %q, want it to contain %q", stderr, wantLine)
	}
	if got := strings.Count(stderr, "Error: "); got != 1 {
		t.Fatalf("stderr = %q, want exactly one %q diagnostic, got %d", stderr, "Error: ", got)
	}
}
