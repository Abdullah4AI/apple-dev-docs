package web

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

var listRemovedWebAppsFn = func(ctx context.Context, client *webcore.Client, opts webcore.RemovedAppsListOptions) (*webcore.RemovedAppsListResponse, error) {
	return client.ListRemovedApps(ctx, opts)
}

// WebRemovedAppsCommand returns the removed apps command group.
func WebRemovedAppsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web removed-apps", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "removed-apps",
		ShortUsage: "asc web removed-apps <subcommand> [flags]",
		ShortHelp:  "Inspect removed apps via web sessions.",
		LongHelp: `WEB SESSION WORKFLOWS

Inspect apps shown in App Store Connect's Removed Apps status view.

Examples:
  asc web removed-apps list
  asc web removed-apps list --paginate --output table
  asc web removed-apps list --apple-id "user@example.com" --output json`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			WebRemovedAppsListCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// WebRemovedAppsListCommand lists removed apps using the internal web API.
func WebRemovedAppsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web removed-apps list", flag.ExitOnError)

	authFlags := bindWebSessionFlags(fs)
	limit := fs.Int("limit", webcore.DefaultRemovedAppsLimit, fmt.Sprintf("Maximum results per page (1-%d)", webcore.MaxRemovedAppsLimit))
	next := fs.String("next", "", "Fetch next page using a links.next URL")
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages (aggregate results)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "asc web removed-apps list [flags]",
		ShortHelp:  "List removed apps via Apple web sessions.",
		LongHelp: `List apps from App Store Connect's Removed Apps status view using the
Apple web-session API.

Examples:
  asc web removed-apps list
  asc web removed-apps list --limit 25 --output table
  asc web removed-apps list --paginate --output json
  asc web removed-apps list --next "<links.next>"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageError("web removed-apps list does not accept positional arguments")
			}
			if *limit < 1 || *limit > webcore.MaxRemovedAppsLimit {
				return shared.UsageError(fmt.Sprintf("web removed-apps list: --limit must be between 1 and %d", webcore.MaxRemovedAppsLimit))
			}
			nextValue := strings.TrimSpace(*next)
			if nextValue != "" && *paginate {
				return shared.UsageError("web removed-apps list: --next cannot be combined with --paginate")
			}
			if err := validateRemovedAppsNextURL(nextValue); err != nil {
				return shared.UsageError("web removed-apps list: " + err.Error())
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, authFlags)
			if err != nil {
				return withWebAuthHint(err, "web removed-apps list")
			}

			result, err := listRemovedWebAppsFn(requestCtx, newWebClientFn(session), webcore.RemovedAppsListOptions{
				Limit:    *limit,
				Next:     nextValue,
				Paginate: *paginate,
			})
			if err != nil {
				return withWebAuthHint(err, "web removed-apps list")
			}

			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error { return renderRemovedAppsTable(result) },
				func() error { return renderRemovedAppsMarkdown(result) },
			)
		},
	}
}

func validateRemovedAppsNextURL(next string) error {
	next = strings.TrimSpace(next)
	if next == "" {
		return nil
	}
	parsed, err := url.Parse(next)
	if err != nil {
		return fmt.Errorf("--next must be a valid URL: %w", err)
	}
	if parsed.IsAbs() {
		if parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, "appstoreconnect.apple.com") {
			return fmt.Errorf("--next must be an App Store Connect web URL")
		}
	} else if parsed.Host != "" {
		return fmt.Errorf("--next must be an App Store Connect web URL")
	}
	if path := parsed.EscapedPath(); path != "/iris/v1/apps" && path != "/apps" {
		return fmt.Errorf("--next must be an App Store Connect web URL")
	}
	if parsed.Query().Get("filter[removed]") != "true" {
		return fmt.Errorf("--next must include filter[removed]=true")
	}
	return nil
}

func removedAppsHeaders() []string {
	return []string{"ID", "Name", "Bundle ID", "Version", "Status", "SKU"}
}

func removedAppsRows(result *webcore.RemovedAppsListResponse) [][]string {
	if result == nil {
		return nil
	}
	rows := make([][]string, 0, len(result.Data))
	for _, app := range result.Data {
		rows = append(rows, []string{
			shared.OrNA(app.ID),
			shared.OrNA(app.Name),
			shared.OrNA(app.BundleID),
			shared.OrNA(app.VersionSummary),
			shared.OrNA(app.Status),
			shared.OrNA(app.SKU),
		})
	}
	return rows
}

func renderRemovedAppsTable(result *webcore.RemovedAppsListResponse) error {
	asc.RenderTable(removedAppsHeaders(), removedAppsRows(result))
	return nil
}

func renderRemovedAppsMarkdown(result *webcore.RemovedAppsListResponse) error {
	asc.RenderMarkdown(removedAppsHeaders(), removedAppsRows(result))
	return nil
}
