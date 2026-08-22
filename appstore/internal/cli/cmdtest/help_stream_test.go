package cmdtest

import (
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

// TestRequestedHelpIsWrittenToStdout pins the discovery contract that
// `asc ... --help` is a successful request whose payload belongs on stdout, so
// `asc builds --help | grep wait` and `asc builds --help > file` keep working.
func TestRequestedHelpIsWrittenToStdout(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "root long help", args: []string{"--help"}, want: "asc <subcommand> [flags]"},
		{name: "root single-dash long help", args: []string{"-help"}, want: "asc <subcommand> [flags]"},
		{name: "root double-dash short help", args: []string{"--h"}, want: "asc <subcommand> [flags]"},
		{name: "root short help with value", args: []string{"-h=true"}, want: "asc <subcommand> [flags]"},
		{name: "root long help with value", args: []string{"--help=false"}, want: "asc <subcommand> [flags]"},
		{name: "group long help", args: []string{"builds", "--help"}, want: "asc builds"},
		{name: "group single-dash long help", args: []string{"builds", "-help"}, want: "asc builds"},
		{name: "group double-dash short help with value", args: []string{"builds", "--h=true"}, want: "asc builds"},
		{name: "group short help", args: []string{"builds", "-h"}, want: "asc builds"},
		{name: "leaf long help", args: []string{"builds", "list", "--help"}, want: "asc builds list"},
		{name: "leaf short help", args: []string{"builds", "list", "-h"}, want: "asc builds list"},
		{name: "flagless group help", args: []string{"versions", "--help"}, want: "asc versions"},
		{name: "root help before subcommand", args: []string{"--help", "builds", "list"}, want: "asc <subcommand> [flags]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var code int
			stdout, stderr := captureOutput(t, func() {
				code = rootcmd.Run(test.args, "1.2.3")
			})

			if code != rootcmd.ExitSuccess {
				t.Fatalf("exit code = %d, want %d", code, rootcmd.ExitSuccess)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr for requested help, got %q", stderr)
			}
			if !strings.Contains(stdout, test.want) {
				t.Fatalf("expected stdout to contain %q, got %q", test.want, stdout)
			}
		})
	}
}

// TestRequestedGroupHelpMatchesBareGroupHelp keeps the help payload byte
// identical across both entry points; only the stream changed.
func TestRequestedGroupHelpMatchesBareGroupHelp(t *testing.T) {
	bare, bareStderr := captureOutput(t, func() {
		if code := rootcmd.Run([]string{"versions"}, "1.2.3"); code != rootcmd.ExitSuccess {
			t.Fatalf("bare group exit code = %d, want %d", code, rootcmd.ExitSuccess)
		}
	})
	if bareStderr != "" {
		t.Fatalf("expected empty stderr for bare group, got %q", bareStderr)
	}

	requested, requestedStderr := captureOutput(t, func() {
		if code := rootcmd.Run([]string{"versions", "--help"}, "1.2.3"); code != rootcmd.ExitSuccess {
			t.Fatalf("requested help exit code = %d, want %d", code, rootcmd.ExitSuccess)
		}
	})
	if requestedStderr != "" {
		t.Fatalf("expected empty stderr for requested help, got %q", requestedStderr)
	}

	if bare != requested {
		t.Fatalf("requested help text differs from bare group help:\nbare=%q\nrequested=%q", bare, requested)
	}
}

// TestUsageErrorHelpStaysOnStderr guards the other half of the contract: help
// printed because the invocation was wrong is a diagnostic, not data.
func TestUsageErrorHelpStaysOnStderr(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "unknown flag", args: []string{"builds", "list", "--nonsense"}, want: "unknown flag `--nonsense`"},
		{name: "unknown subcommand", args: []string{"builds", "nonsense"}, want: "unknown command"},
		{name: "bad flag syntax", args: []string{"builds", "list", "---limit"}, want: "bad flag syntax"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var code int
			stdout, stderr := captureOutput(t, func() {
				code = rootcmd.Run(test.args, "1.2.3")
			})

			if code != rootcmd.ExitUsage {
				t.Fatalf("exit code = %d, want %d", code, rootcmd.ExitUsage)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout for usage error, got %q", stdout)
			}
			if !strings.Contains(stderr, test.want) {
				t.Fatalf("expected stderr to contain %q, got %q", test.want, stderr)
			}
		})
	}
}
