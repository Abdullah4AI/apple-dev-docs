package cmdtest

import (
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// TestExperimentalCommandsHaveStabilityLabel ensures every command surface
// that is marked experimental carries a consistent "[experimental]" prefix in
// its ShortHelp so that the label is visible in grouped root help, subcommand
// listings, and generated docs.
func TestExperimentalCommandsHaveStabilityLabel(t *testing.T) {
	root := RootCommand("1.2.3")

	cases := []struct {
		path []string // subcommand path from root
	}{
		{[]string{"system-status"}},
		{[]string{"screenshots", "run"}},
		{[]string{"screenshots", "capture"}},
		{[]string{"screenshots", "frame"}},
		{[]string{"screenshots", "list-frame-devices"}},
		{[]string{"screenshots", "review-generate"}},
		{[]string{"screenshots", "review-open"}},
		{[]string{"screenshots", "review-approve"}},
		{[]string{"screenshots", "plan"}},
		{[]string{"screenshots", "apply"}},
		{[]string{"xcode", "build"}},
		{[]string{"signing", "reconcile"}},
		{[]string{"signing", "reconcile", "plan"}},
		{[]string{"signing", "reconcile", "apply"}},
		{[]string{"apps", "rename"}},
		{[]string{"web", "apps", "transfer"}},
		{[]string{"web", "apps", "transfer", "status"}},
		{[]string{"web", "agreements"}},
		{[]string{"web", "agreements", "status"}},
		{[]string{"web", "agreements", "accept"}},
		{[]string{"web", "auth", "export"}},
		{[]string{"web", "auth", "import"}},
		{[]string{"web", "bundle-ids", "list"}},
		{[]string{"web", "bundle-ids", "view"}},
		{[]string{"web", "service-ids"}},
		{[]string{"web", "service-ids", "list"}},
		{[]string{"web", "service-ids", "view"}},
		{[]string{"web", "service-ids", "create"}},
		{[]string{"web", "service-ids", "rename"}},
		{[]string{"web", "service-ids", "delete"}},
		{[]string{"web", "review", "reply"}},
		{[]string{"web", "review", "drafts"}},
		{[]string{"web", "review", "drafts", "create"}},
		{[]string{"web", "review", "drafts", "update"}},
		{[]string{"web", "review", "drafts", "delete"}},
		{[]string{"web", "website-push-ids"}},
		{[]string{"web", "website-push-ids", "list"}},
		{[]string{"web", "website-push-ids", "view"}},
		{[]string{"web", "website-push-ids", "create"}},
		{[]string{"web", "website-push-ids", "delete"}},
		{[]string{"web", "icloud-containers"}},
		{[]string{"web", "icloud-containers", "list"}},
		{[]string{"web", "finance"}},
		{[]string{"web", "finance", "transaction-tax"}},
		{[]string{"web", "finance", "transaction-tax", "download"}},
		{[]string{"web", "iap"}},
		{[]string{"web", "iap", "tax-category"}},
		{[]string{"web", "iap", "tax-category", "list"}},
		{[]string{"web", "iap", "tax-category", "view"}},
		{[]string{"web", "iap", "tax-category", "set"}},
		{[]string{"web", "iap", "tax-category", "reset"}},
		{[]string{"web", "apps", "distribution", "users"}},
		{[]string{"web", "apps", "distribution", "users", "list"}},
		{[]string{"web", "apps", "distribution", "users", "create"}},
		{[]string{"web", "apps", "distribution", "users", "delete"}},
		{[]string{"web", "xcode-cloud", "scm"}},
		{[]string{"web", "xcode-cloud", "scm", "providers"}},
		{[]string{"web", "xcode-cloud", "scm", "providers", "list"}},
		{[]string{"web", "xcode-cloud", "scm", "connection-status"}},
	}

	for _, tc := range cases {
		cmd := findSubcommand(root, tc.path...)
		assertExperimentalCommand(t, cmd, tc.path)
	}
}

func TestScreenshotsParentHelpRetainsLocalExperimentalScope(t *testing.T) {
	root := RootCommand("1.2.3")
	cmd := findSubcommand(root, "screenshots")
	if cmd == nil {
		t.Fatal("command [screenshots] not found")
	}
	if !strings.Contains(cmd.ShortHelp, "local capture/frame/matrix workflow is [experimental]") {
		t.Fatalf("screenshots ShortHelp = %q, want broad local experimental marker", cmd.ShortHelp)
	}
	if !strings.Contains(cmd.LongHelp, "Local screenshot automation commands are experimental.") {
		t.Fatalf("screenshots LongHelp = %q, want broad local experimental warning", cmd.LongHelp)
	}
}

func TestWebCommandsDoNotHaveExperimentalStabilityLabel(t *testing.T) {
	root := RootCommand("1.2.3")

	webCmd := findSubcommand(root, "web")
	if webCmd == nil {
		t.Fatal("command [web] not found")
	}
	assertCommandDoesNotMentionExperimental(t, webCmd, []string{"web"})
	allowed := map[string]struct{}{
		"web apps transfer":           {},
		"web apps transfer status":    {},
		"web auth export":             {},
		"web auth import":             {},
		"web bundle-ids list":         {},
		"web bundle-ids view":         {},
		"web service-ids":             {},
		"web service-ids list":        {},
		"web service-ids view":        {},
		"web service-ids create":      {},
		"web service-ids rename":      {},
		"web service-ids delete":      {},
		"web review reply":            {},
		"web review drafts":           {},
		"web review drafts create":    {},
		"web review drafts update":    {},
		"web review drafts delete":    {},
		"web website-push-ids":        {},
		"web website-push-ids list":   {},
		"web website-push-ids view":   {},
		"web website-push-ids create": {},
		"web website-push-ids delete": {},

		"web icloud-containers":                {},
		"web icloud-containers list":           {},
		"web finance":                          {},
		"web finance transaction-tax":          {},
		"web finance transaction-tax download": {},
		"web iap":                              {},
		"web iap tax-category":                 {},
		"web iap tax-category list":            {},
		"web iap tax-category view":            {},
		"web iap tax-category set":             {},
		"web iap tax-category reset":           {},
		"web apps distribution users":          {},
		"web apps distribution users list":     {},
		"web apps distribution users create":   {},
		"web apps distribution users delete":   {},

		"web xcode-cloud":                       {},
		"web xcode-cloud scm":                   {},
		"web xcode-cloud scm providers":         {},
		"web xcode-cloud scm providers list":    {},
		"web xcode-cloud scm connection-status": {},
	}
	for _, sub := range webCmd.Subcommands {
		if sub.Name == "agreements" {
			continue
		}
		assertCommandTreeDoesNotMentionExperimentalExcept(t, sub, []string{"web", sub.Name}, allowed)
	}
}

func TestWebCommandsDoNotHaveEndpointWarningLabels(t *testing.T) {
	root := RootCommand("1.2.3")

	webCmd := findSubcommand(root, "web")
	if webCmd == nil {
		t.Fatal("command [web] not found")
	}
	assertCommandTreeDoesNotMentionEndpointWarnings(t, webCmd, []string{"web"})
}

func assertCommandTreeDoesNotMentionExperimentalExcept(t *testing.T, cmd *ffcli.Command, path []string, allowed map[string]struct{}) {
	t.Helper()

	if _, ok := allowed[strings.Join(path, " ")]; !ok {
		assertCommandDoesNotMentionExperimental(t, cmd, path)
	}
	for _, sub := range cmd.Subcommands {
		assertCommandTreeDoesNotMentionExperimentalExcept(t, sub, append(path, sub.Name), allowed)
	}
}

func assertExperimentalCommand(t *testing.T, cmd *ffcli.Command, path []string) {
	t.Helper()

	if cmd == nil {
		t.Errorf("command %v not found", path)
		return
	}
	if !strings.HasPrefix(cmd.ShortHelp, "[experimental]") {
		t.Errorf("command %v: expected ShortHelp to start with [experimental], got %q", path, cmd.ShortHelp)
	}
}

func assertCommandDoesNotMentionExperimental(t *testing.T, cmd *ffcli.Command, path []string) {
	t.Helper()

	if cmd == nil {
		t.Errorf("command %v not found", path)
		return
	}
	if strings.Contains(strings.ToLower(cmd.ShortHelp), "experimental") {
		t.Errorf("command %v: expected ShortHelp not to mention experimental, got %q", path, cmd.ShortHelp)
	}
	if strings.Contains(strings.ToLower(cmd.LongHelp), "experimental") {
		t.Errorf("command %v: expected LongHelp not to mention experimental, got %q", path, cmd.LongHelp)
	}
}

func assertCommandTreeDoesNotMentionEndpointWarnings(t *testing.T, cmd *ffcli.Command, path []string) {
	t.Helper()

	assertCommandDoesNotMentionEndpointWarnings(t, cmd, path)

	for _, sub := range cmd.Subcommands {
		assertCommandTreeDoesNotMentionEndpointWarnings(t, sub, append(path, sub.Name))
	}
}

func assertCommandDoesNotMentionEndpointWarnings(t *testing.T, cmd *ffcli.Command, path []string) {
	t.Helper()

	if cmd == nil {
		t.Errorf("command %v not found", path)
		return
	}
	help := strings.ToLower(cmd.ShortHelp + "\n" + cmd.LongHelp)
	for _, token := range []string{
		"unofficial",
		"discouraged",
		"private endpoint",
		"private web",
		"not sanctioned",
		"at your own risk",
		"account restrictions",
		"production-critical",
		"break without notice",
	} {
		if strings.Contains(help, token) {
			t.Errorf("command %v: expected help not to mention %q, got %q", path, token, cmd.LongHelp)
		}
	}
}
