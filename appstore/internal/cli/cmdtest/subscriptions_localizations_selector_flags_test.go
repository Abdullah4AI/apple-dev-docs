package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

const (
	subscriptionsLocalizationsUpdateDeprecationWarning = "Warning: `asc subscriptions localizations update` is deprecated by App Store Connect API 4.4.1. Use `asc subscriptions versions localizations update --id \"LOCALIZATION_ID\" --name \"NAME\"`."
	subscriptionsLocalizationsProductIDAliasWarning    = "Warning: `--product-id` is deprecated. Use `--subscription-id`."
)

func failingTransport(t *testing.T) {
	t.Helper()

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("expected no HTTP request, got %s %s", req.Method, req.URL.String())
		return nil, nil
	})
}

func subscriptionsLocalizationsCommand(t *testing.T, root *ffcli.Command, name string) *ffcli.Command {
	t.Helper()

	command := findSubcommand(root, "subscriptions", "localizations", name)
	if command == nil {
		t.Fatalf("command %q not found", "subscriptions localizations "+name)
	}
	return command
}

func runSubscriptionsLocalizationsArgs(t *testing.T, root *ffcli.Command, args []string) (string, string, error) {
	t.Helper()

	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse(args); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	return stdout, stderr, runErr
}

func TestSubscriptionsLocalizationsCreateProductIDAliasResolvesSubscription(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	root := RootCommand("1.2.3")
	create := subscriptionsLocalizationsCommand(t, root, "create")
	if create.FlagSet.Lookup("product-id") == nil {
		t.Fatal("expected --product-id compatibility alias on subscriptions localizations create")
	}

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if req.Method != http.MethodGet || req.URL.Path != "/v1/subscriptions/123456789/subscriptionLocalizations" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"subscriptionLocalizations","id":"loc-1","attributes":{"name":"Pro","locale":"en-US","description":"Premium features."}}],"links":{"next":""}}`), nil
	})

	stdout, stderr, runErr := runSubscriptionsLocalizationsArgs(t, root, []string{
		"subscriptions", "localizations", "create",
		"--product-id", "123456789",
		"--locale", "en-us",
		"--name", "Pro",
		"--description", "Premium features.",
		"--output", "json",
	})
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	if requestCount != 1 {
		t.Fatalf("expected the alias to select the same subscription, got %d requests", requestCount)
	}

	requireStderrContainsWarning(t, stderr, subscriptionsLocalizationsCreateDeprecationWarning)
	requireStderrContainsWarning(t, stderr, subscriptionsLocalizationsProductIDAliasWarning)

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("parse output: %v\nstdout=%q", err, stdout)
	}
	if result.Data.ID != "loc-1" {
		t.Fatalf("unexpected localization output: %+v", result)
	}
}

func TestSubscriptionsLocalizationsCreateProductIDAliasConflictErrors(t *testing.T) {
	failingTransport(t)

	root := RootCommand("1.2.3")
	stdout, stderr, runErr := runSubscriptionsLocalizationsArgs(t, root, []string{
		"subscriptions", "localizations", "create",
		"--subscription-id", "123456789",
		"--product-id", "com.example.pro",
		"--locale", "en-US",
		"--name", "Pro",
	})
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --product-id conflicts with --subscription-id; use only --subscription-id") {
		t.Fatalf("expected conflicting subscription selector error, got %q", stderr)
	}
}

func TestSubscriptionsLocalizationsCreateRejectsVersionIDWithVersionScopedGuidance(t *testing.T) {
	failingTransport(t)

	root := RootCommand("1.2.3")
	create := subscriptionsLocalizationsCommand(t, root, "create")
	if create.FlagSet.Lookup("version-id") == nil {
		t.Fatal("expected --version-id to be recognized for guidance on subscriptions localizations create")
	}

	stdout, stderr, runErr := runSubscriptionsLocalizationsArgs(t, root, []string{
		"subscriptions", "localizations", "create",
		"--version-id", "SUBSCRIPTION_VERSION_ID",
		"--locale", "en-US",
		"--name", "Pro",
	})
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--version-id is not accepted here") {
		t.Fatalf("expected version selector rejection, got %q", stderr)
	}
	if !strings.Contains(stderr, "--subscription-id") {
		t.Fatalf("expected product-scoped selector guidance, got %q", stderr)
	}
	if !strings.Contains(stderr, "asc subscriptions versions localizations create --version-id") {
		t.Fatalf("expected version-scoped command guidance, got %q", stderr)
	}
	if got := create.FlagSet.Lookup("subscription-id").Value.String(); got != "" {
		t.Fatalf("expected --version-id never to populate --subscription-id, got %q", got)
	}
}

func TestSubscriptionsLocalizationsUpdateRejectsSubscriptionIDWithLocalizationGuidance(t *testing.T) {
	failingTransport(t)

	root := RootCommand("1.2.3")
	update := subscriptionsLocalizationsCommand(t, root, "update")
	if update.FlagSet.Lookup("subscription-id") == nil {
		t.Fatal("expected --subscription-id to be recognized for guidance on subscriptions localizations update")
	}

	stdout, stderr, runErr := runSubscriptionsLocalizationsArgs(t, root, []string{
		"subscriptions", "localizations", "update",
		"--subscription-id", "123456789",
		"--name", "Pro+",
	})
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--subscription-id is not accepted here") {
		t.Fatalf("expected subscription selector rejection, got %q", stderr)
	}
	if !strings.Contains(stderr, "--id is the localization ID, not the subscription ID") {
		t.Fatalf("expected localization identity explanation, got %q", stderr)
	}
	if !strings.Contains(stderr, "asc subscriptions localizations list --subscription-id") {
		t.Fatalf("expected localization discovery guidance, got %q", stderr)
	}
	if got := update.FlagSet.Lookup("id").Value.String(); got != "" {
		t.Fatalf("expected --subscription-id never to populate --id, got %q", got)
	}
}

func TestSubscriptionsLocalizationsUpdateSubscriptionIDGuidanceOutranksMissingID(t *testing.T) {
	failingTransport(t)

	root := RootCommand("1.2.3")
	_, stderr, runErr := runSubscriptionsLocalizationsArgs(t, root, []string{
		"subscriptions", "localizations", "update",
		"--subscription-id", "123456789",
	})
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp usage error, got %v", runErr)
	}
	if strings.Contains(stderr, "Error: --id is required") {
		t.Fatalf("expected selector guidance instead of a bare missing-flag error, got %q", stderr)
	}
	if !strings.Contains(stderr, "--subscription-id is not accepted here") {
		t.Fatalf("expected subscription selector rejection, got %q", stderr)
	}
}

func TestSubscriptionsLocalizationsCompatibilityFlagsStayHiddenFromHelp(t *testing.T) {
	root := RootCommand("1.2.3")

	tests := []struct {
		command string
		flags   []string
	}{
		{command: "create", flags: []string{"product-id", "version-id"}},
		{command: "update", flags: []string{"subscription-id"}},
	}

	for _, test := range tests {
		command := subscriptionsLocalizationsCommand(t, root, test.command)
		usage := command.UsageFunc(command)
		for _, name := range test.flags {
			if command.FlagSet.Lookup(name) == nil {
				t.Fatalf("expected subscriptions localizations %s to recognize --%s", test.command, name)
			}
			if strings.Contains(usage, "\n  --"+name+" ") {
				t.Fatalf("expected --%s to stay hidden from subscriptions localizations %s help: %q", name, test.command, usage)
			}
		}
	}
}

func TestSubscriptionsLocalizationsCanonicalSelectorsStayQuiet(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch || req.URL.Path != "/v1/subscriptionLocalizations/loc-1" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"subscriptionLocalizations","id":"loc-1","attributes":{"name":"Pro+","locale":"en-US","description":"Premium features."}}}`), nil
	})

	root := RootCommand("1.2.3")
	_, stderr, runErr := runSubscriptionsLocalizationsArgs(t, root, []string{
		"subscriptions", "localizations", "update",
		"--id", "loc-1",
		"--name", "Pro+",
		"--output", "json",
	})
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	assertOnlyCommandDeprecationWarning(t, stderr, subscriptionsLocalizationsUpdateDeprecationWarning)
}
