package web

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

func stubWebAppsDistributionSession(t *testing.T) {
	t.Helper()

	origResolveSession := resolveSessionFn
	origNewWebClient := newWebClientFn
	t.Cleanup(func() {
		resolveSessionFn = origResolveSession
		newWebClientFn = origNewWebClient
	})

	resolveSessionFn = func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{}, "cache", nil
	}
	newWebClientFn = func(session *webcore.AuthSession) *webcore.Client {
		return &webcore.Client{}
	}
}

func TestWebAppsDistributionViewPrintsAppleAttributes(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	stubWebAppsDistributionSession(t)

	origGet := getWebAppDistributionFn
	t.Cleanup(func() { getWebAppDistributionFn = origGet })

	var gotAppID string
	getWebAppDistributionFn = func(ctx context.Context, client *webcore.Client, appID string) (*webcore.AppDistribution, error) {
		gotAppID = appID
		return &webcore.AppDistribution{
			AppID:                 appID,
			Name:                  "Example",
			BundleID:              "com.example.app",
			DistributionType:      "CUSTOM",
			EducationDiscountType: "NOT_APPLICABLE",
		}, nil
	}

	cmd := WebAppsDistributionViewCommand()
	if err := cmd.FlagSet.Parse([]string{"--app", "app-1", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, stderr := captureWebCommandOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if gotAppID != "app-1" {
		t.Fatalf("appID = %q, want app-1", gotAppID)
	}

	var out struct {
		AppID                 string `json:"appId"`
		BundleID              string `json:"bundleId"`
		DistributionType      string `json:"distributionType"`
		EducationDiscountType string `json:"educationDiscountType"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v; stdout=%q", err, stdout)
	}
	if out.AppID != "app-1" || out.BundleID != "com.example.app" {
		t.Fatalf("unexpected identity fields: %+v", out)
	}
	if out.DistributionType != "CUSTOM" {
		t.Fatalf("distributionType = %q, want CUSTOM", out.DistributionType)
	}
	if out.EducationDiscountType != "NOT_APPLICABLE" {
		t.Fatalf("educationDiscountType = %q, want NOT_APPLICABLE", out.EducationDiscountType)
	}
}

func TestWebAppsDistributionViewTableRendersUnknownForMissingAttributes(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	stubWebAppsDistributionSession(t)

	origGet := getWebAppDistributionFn
	t.Cleanup(func() { getWebAppDistributionFn = origGet })

	getWebAppDistributionFn = func(ctx context.Context, client *webcore.Client, appID string) (*webcore.AppDistribution, error) {
		return &webcore.AppDistribution{AppID: appID}, nil
	}

	cmd := WebAppsDistributionViewCommand()
	if err := cmd.FlagSet.Parse([]string{"--app", "app-2", "--output", "table"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, stderr := captureWebCommandOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"distribution_type", "unknown", "education_discount_type", "app-2"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q; stdout=%q", want, stdout)
		}
	}
}

func TestWebAppsDistributionViewRequiresApp(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	cmd := WebAppsDistributionViewCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var err error
	_, stderr := captureWebCommandOutput(t, func() {
		err = cmd.Exec(context.Background(), nil)
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if want := "Error: --app is required (or set ASC_APP_ID)\n"; !strings.HasPrefix(stderr, want) {
		t.Fatalf("stderr = %q, want prefix %q", stderr, want)
	}
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v, want flag.ErrHelp usage contract", err)
	}
	if kind := shared.ClassifyUsageError(err); kind != shared.UsageErrorMissingRequired {
		t.Fatalf("usage kind = %q, want %q", kind, shared.UsageErrorMissingRequired)
	}
}

func TestWebAppsDistributionViewRejectsPositionalArguments(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	cmd := WebAppsDistributionViewCommand()
	if err := cmd.FlagSet.Parse([]string{"--app", "app-1"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	err := cmd.Exec(context.Background(), []string{"extra"})
	if err == nil {
		t.Fatal("expected error for positional argument")
	}
	if !strings.Contains(err.Error(), "unexpected argument") {
		t.Fatalf("error = %v, want unexpected argument", err)
	}
}

func TestWebAppsDistributionGroupIsRegisteredUnderWebApps(t *testing.T) {
	group := WebAppsDistributionCommand()
	if group.Name != "distribution" {
		t.Fatalf("group name = %q, want distribution", group.Name)
	}
	if len(group.Subcommands) == 0 {
		t.Fatal("expected distribution subcommands")
	}

	var hasDistribution bool
	for _, sub := range WebAppsCommand().Subcommands {
		if sub.Name == "distribution" {
			hasDistribution = true
		}
	}
	if !hasDistribution {
		t.Fatal("expected asc web apps distribution to be registered")
	}
}
