package web

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

var getWebAppDistributionFn = func(ctx context.Context, client *webcore.Client, appID string) (*webcore.AppDistribution, error) {
	return client.GetAppDistribution(ctx, appID)
}

// WebAppsDistributionCommand returns the web app distribution method command group.
func WebAppsDistributionCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps distribution", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "distribution",
		ShortUsage: "asc web apps distribution <subcommand> [flags]",
		ShortHelp:  "Inspect the app distribution method via web sessions.",
		LongHelp: `WEB SESSION WORKFLOWS

Read the app-level distribution method that App Store Connect shows under
Distribution -> App Availability. The public App Store Connect API does not
expose this setting, so it is only reachable through a web session.

This command is read-only. Changing the distribution method and requesting
unlisted distribution have Apple-side eligibility restrictions whose write
contracts are not captured yet, so keep using the App Store Connect web UI for
those changes.

`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			WebAppsDistributionViewCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// WebAppsDistributionViewCommand returns the distribution method view command.
func WebAppsDistributionViewCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps distribution view", flag.ExitOnError)

	appID := fs.String("app", "", "App Store Connect app ID (or ASC_APP_ID)")
	authFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "view",
		ShortUsage: "asc web apps distribution view --app APP_ID [flags]",
		ShortHelp:  "View the app distribution method.",
		LongHelp: `WEB SESSION WORKFLOWS

View the app-level distribution method attributes returned by Apple's internal
app resource. Values are printed exactly as Apple returns them; APP_STORE is
public App Store distribution and CUSTOM is private distribution through Apple
Business Manager or Apple School Manager. Attributes Apple omits are reported as
"unknown" in table output.

Examples:
  asc web apps distribution view --app 6759231657
  asc web apps distribution view --app 6759231657 --output json`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageErrorf("unexpected argument(s): %s", strings.Join(args, " "))
			}

			resolvedAppID := strings.TrimSpace(shared.ResolveAppID(*appID))
			if resolvedAppID == "" {
				fmt.Fprintln(os.Stderr, "Error: --app is required (or set ASC_APP_ID)")
				return shared.MissingRequiredUsageError("--app")
			}

			session, requestCtx, cancel, err := resolveWebSessionForCommand(ctx, authFlags)
			defer cancel()
			if err != nil {
				return err
			}

			var result *webcore.AppDistribution
			err = withWebSpinner("Fetching app distribution method", func() error {
				var err error
				result, err = getWebAppDistributionFn(requestCtx, newWebClientFn(session), resolvedAppID)
				return err
			})
			if err != nil {
				return withWebAuthHint(err, "web apps distribution view")
			}

			return printWebAppDistribution(result, *output.Output, *output.Pretty)
		},
	}
}

func printWebAppDistribution(result *webcore.AppDistribution, output string, pretty bool) error {
	return shared.PrintOutputWithRenderers(
		result,
		output,
		pretty,
		func() error {
			asc.RenderTable([]string{"field", "value"}, webAppDistributionRows(result))
			return nil
		},
		func() error {
			asc.RenderMarkdown([]string{"field", "value"}, webAppDistributionRows(result))
			return nil
		},
	)
}

func webAppDistributionRows(result *webcore.AppDistribution) [][]string {
	if result == nil {
		return nil
	}
	return [][]string{
		{"app_id", result.AppID},
		{"name", webAppValueOrUnknown(result.Name)},
		{"bundle_id", webAppValueOrUnknown(result.BundleID)},
		{"distribution_type", webAppValueOrUnknown(result.DistributionType)},
		{"education_discount_type", webAppValueOrUnknown(result.EducationDiscountType)},
	}
}
