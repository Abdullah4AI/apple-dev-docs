package cmdtest

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

// Hidden compatibility flag spellings on the subscriptions commands were
// removed in 5.0.0. They now fail as generic unknown flags (exit 2) before any
// client is resolved, and the canonical spellings are the only bound flags.
func TestSubscriptionsRemovedFlagAliasesAreUnknownFlags(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_APP_ID", "")

	stubTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("removed flag must fail before HTTP: %s %s", req.Method, req.URL.String())
		return nil, errors.New("unexpected request")
	}))

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "list version-limit",
			args:    []string{"subscriptions", "list", "--group-id", "group-1", "--version-limit", "1"},
			wantErr: "Error: unknown flag `--version-limit` for `asc subscriptions list`",
		},
		{
			name:    "view version-limit",
			args:    []string{"subscriptions", "view", "--id", "sub-1", "--version-limit", "1"},
			wantErr: "Error: unknown flag `--version-limit` for `asc subscriptions view`",
		},
		{
			name:    "view subscription-id",
			args:    []string{"subscriptions", "view", "--subscription-id", "sub-1"},
			wantErr: "Error: unknown flag `--subscription-id` for `asc subscriptions view`",
		},
		{
			name:    "versions list image-limit",
			args:    []string{"subscriptions", "versions", "list", "--subscription-id", "sub-1", "--image-limit", "1"},
			wantErr: "Error: unknown flag `--image-limit` for `asc subscriptions versions list`",
		},
		{
			name:    "versions list localization-limit",
			args:    []string{"subscriptions", "versions", "list", "--subscription-id", "sub-1", "--localization-limit", "1"},
			wantErr: "Error: unknown flag `--localization-limit` for `asc subscriptions versions list`",
		},
		{
			name:    "versions view image-limit",
			args:    []string{"subscriptions", "versions", "view", "--id", "version-1", "--image-limit", "1"},
			wantErr: "Error: unknown flag `--image-limit` for `asc subscriptions versions view`",
		},
		{
			name:    "versions view localization-limit",
			args:    []string{"subscriptions", "versions", "view", "--id", "version-1", "--localization-limit", "1"},
			wantErr: "Error: unknown flag `--localization-limit` for `asc subscriptions versions view`",
		},
		{
			name:    "review screenshots delete id",
			args:    []string{"subscriptions", "review", "screenshots", "delete", "--id", "shot-1", "--confirm"},
			wantErr: "Error: unknown flag `--id` for `asc subscriptions review screenshots delete`",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				if code := rootcmd.Run(test.args, "5.0.0"); code != rootcmd.ExitUsage {
					t.Fatalf("exit code = %d, want %d", code, rootcmd.ExitUsage)
				}
			})
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("stderr = %q, want unknown-flag error %q", stderr, test.wantErr)
			}
			if strings.Contains(stderr, "deprecated") {
				t.Fatalf("stderr = %q, want no deprecation guidance for a removed flag", stderr)
			}
		})
	}
}

func TestSubscriptionsRemovedFlagAliasesAreNotBound(t *testing.T) {
	root := RootCommand("5.0.0")
	tests := []struct {
		path      []string
		removed   string
		canonical string
	}{
		{path: []string{"subscriptions", "list"}, removed: "version-limit", canonical: "versions-limit"},
		{path: []string{"subscriptions", "view"}, removed: "version-limit", canonical: "versions-limit"},
		{path: []string{"subscriptions", "view"}, removed: "subscription-id", canonical: "id"},
		{path: []string{"subscriptions", "versions", "list"}, removed: "image-limit", canonical: "images-limit"},
		{path: []string{"subscriptions", "versions", "list"}, removed: "localization-limit", canonical: "localizations-limit"},
		{path: []string{"subscriptions", "versions", "view"}, removed: "image-limit", canonical: "images-limit"},
		{path: []string{"subscriptions", "versions", "view"}, removed: "localization-limit", canonical: "localizations-limit"},
		{path: []string{"subscriptions", "review", "screenshots", "delete"}, removed: "id", canonical: "screenshot-id"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test.path, " ")+" --"+test.removed, func(t *testing.T) {
			command := findSubcommand(root, test.path...)
			if command == nil {
				t.Fatalf("command %q not found", strings.Join(test.path, " "))
			}
			if command.FlagSet.Lookup(test.canonical) == nil {
				t.Fatalf("canonical flag --%s not found", test.canonical)
			}
			if command.FlagSet.Lookup(test.removed) != nil {
				t.Fatalf("removed alias --%s is still bound", test.removed)
			}
		})
	}
}
