package subscriptions

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

type relationshipLimitCommandCase struct {
	name      string
	command   func() *ffcli.Command
	baseArgs  []string
	canonical []string
	removed   []string
}

func relationshipLimitCommandCases() []relationshipLimitCommandCase {
	return []relationshipLimitCommandCase{
		{
			name: "subscriptions list", command: SubscriptionsListCommand,
			baseArgs:  []string{"--group-id", "group-1"},
			canonical: []string{"versions-limit"}, removed: []string{"version-limit"},
		},
		{
			name: "subscriptions view", command: SubscriptionsGetCommand,
			baseArgs:  []string{"--id", "sub-1"},
			canonical: []string{"versions-limit"}, removed: []string{"version-limit"},
		},
		{
			name: "subscription versions list", command: SubscriptionsVersionsListCommand,
			baseArgs:  []string{"--subscription-id", "sub-1"},
			canonical: []string{"images-limit", "localizations-limit"}, removed: []string{"image-limit", "localization-limit"},
		},
		{
			name: "subscription versions view", command: SubscriptionsVersionsViewCommand,
			baseArgs:  []string{"--id", "version-1"},
			canonical: []string{"images-limit", "localizations-limit"}, removed: []string{"image-limit", "localization-limit"},
		},
	}
}

func captureRelationshipLimitStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	os.Stderr = writer
	defer func() { os.Stderr = old }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		_ = reader.Close()
		done <- buf.String()
	}()

	fn()
	_ = writer.Close()
	return <-done
}

// The singular relationship-limit spellings were removed in 5.0.0. Only the
// plural canonical flags remain bound, so the removed spellings fall through to
// the generic unknown-flag parse error.
func TestSubscriptionRelationshipLimitsExposeOnlyPluralCanonicalFlags(t *testing.T) {
	for _, test := range relationshipLimitCommandCases() {
		t.Run(test.name, func(t *testing.T) {
			cmd := test.command()
			usage := shared.DefaultUsageFunc(cmd)
			for index, canonical := range test.canonical {
				removed := test.removed[index]
				if cmd.FlagSet.Lookup(canonical) == nil {
					t.Fatalf("expected --%s", canonical)
				}
				if !strings.Contains(usage, "--"+canonical) {
					t.Fatalf("canonical help does not contain --%s: %q", canonical, usage)
				}
				if cmd.FlagSet.Lookup(removed) != nil {
					t.Fatalf("removed alias --%s is still bound", removed)
				}
				if strings.Contains(usage, "--"+removed) {
					t.Fatalf("canonical help still mentions removed --%s: %q", removed, usage)
				}
			}
		})
	}
}

func TestSubscriptionRelationshipLimitRemovedAliasesFailToParse(t *testing.T) {
	for _, test := range relationshipLimitCommandCases() {
		for _, removed := range test.removed {
			t.Run(test.name+" "+removed, func(t *testing.T) {
				cmd := test.command()
				cmd.FlagSet.Init(cmd.FlagSet.Name(), flag.ContinueOnError)
				cmd.FlagSet.SetOutput(io.Discard)
				args := append(append([]string{}, test.baseArgs...), "--"+removed, "7")
				err := cmd.FlagSet.Parse(args)
				if err == nil || !strings.Contains(err.Error(), "flag provided but not defined: -"+removed) {
					t.Fatalf("Parse() error = %v, want unknown flag -%s", err, removed)
				}
			})
		}
	}
}

func TestSubscriptionRelationshipLimitsCanonicalFlagsReachClientSetup(t *testing.T) {
	isolateSubscriptionsAuthEnv(t)
	for _, test := range relationshipLimitCommandCases() {
		t.Run(test.name, func(t *testing.T) {
			cmd := test.command()
			args := append([]string{}, test.baseArgs...)
			for index, canonical := range test.canonical {
				args = append(args, "--"+canonical, string(rune('7'+index)))
			}
			if err := cmd.FlagSet.Parse(args); err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			stderr := captureRelationshipLimitStderr(t, func() {
				err := cmd.Exec(context.Background(), nil)
				if err == nil {
					t.Fatal("expected isolated client setup to fail")
				}
				if errors.Is(err, flag.ErrHelp) {
					t.Fatalf("canonical flags failed usage validation: %v", err)
				}
			})
			if strings.Contains(stderr, "deprecated") {
				t.Fatalf("canonical flags emitted a deprecation warning: %q", stderr)
			}
		})
	}
}

func TestSubscriptionRelationshipLimitsRejectInvalidCanonicalValues(t *testing.T) {
	for _, test := range relationshipLimitCommandCases() {
		for _, canonical := range test.canonical {
			t.Run(test.name+" "+canonical, func(t *testing.T) {
				cmd := test.command()
				args := append(append([]string{}, test.baseArgs...), "--"+canonical, "51")
				if err := cmd.FlagSet.Parse(args); err != nil {
					t.Fatalf("Parse() error: %v", err)
				}
				err := cmd.Exec(context.Background(), nil)
				if !errors.Is(err, flag.ErrHelp) || !strings.Contains(err.Error(), "--"+canonical+" must be between 1 and 50") {
					t.Fatalf("Exec() error = %v, want canonical range usage error", err)
				}
			})
		}
	}
}

func TestSubscriptionRelationshipLimitsConflictWithOpaqueNext(t *testing.T) {
	tests := []struct {
		name      string
		command   func() *ffcli.Command
		canonical string
	}{
		{name: "subscription versions", command: SubscriptionsListCommand, canonical: "versions-limit"},
		{name: "version images", command: SubscriptionsVersionsListCommand, canonical: "images-limit"},
		{name: "version localizations", command: SubscriptionsVersionsListCommand, canonical: "localizations-limit"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := test.command()
			if err := cmd.FlagSet.Parse([]string{"--next", "https://api.appstoreconnect.apple.com/v1/example?cursor=next", "--" + test.canonical, "7"}); err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			stderr := captureRelationshipLimitStderr(t, func() {
				err := cmd.Exec(context.Background(), nil)
				if !errors.Is(err, flag.ErrHelp) || !strings.Contains(err.Error(), "--next cannot be combined with --"+test.canonical) {
					t.Fatalf("Exec() error = %v, want canonical opaque-next conflict", err)
				}
			})
			if strings.Contains(stderr, "Warning:") {
				t.Fatalf("unexpected warning: %q", stderr)
			}
		})
	}
}
