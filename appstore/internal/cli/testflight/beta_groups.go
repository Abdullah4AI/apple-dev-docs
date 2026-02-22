package testflight

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// BetaGroupsCommand returns the beta groups command with subcommands.
func BetaGroupsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("beta-groups", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "beta-groups",
		ShortUsage: "appstore testflight beta-groups <subcommand> [flags]",
		ShortHelp:  "Manage TestFlight beta groups.",
		LongHelp: `Manage TestFlight beta groups.

Examples:
  appstore testflight beta-groups list --app "APP_ID"
  appstore testflight beta-groups list --app "APP_ID" --internal
  appstore testflight beta-groups list --global --internal
  appstore testflight beta-groups create --app "APP_ID" --name "Beta Testers"
  appstore testflight beta-groups create --app "APP_ID" --name "Internal Testers" --internal
  appstore testflight beta-groups app get --group-id "GROUP_ID"
  appstore testflight beta-groups beta-recruitment-criteria get --group-id "GROUP_ID"
  appstore testflight beta-groups beta-recruitment-criterion-compatible-build-check get --group-id "GROUP_ID"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			BetaGroupsListCommand(),
			BetaGroupsCreateCommand(),
			BetaGroupsGetCommand(),
			BetaGroupsAppCommand(),
			BetaGroupsRecruitmentCriteriaCommand(),
			BetaGroupsRecruitmentCriterionCompatibleBuildCheckCommand(),
			BetaGroupsUpdateCommand(),
			BetaGroupsAddTestersCommand(),
			BetaGroupsRemoveTestersCommand(),
			BetaGroupsRelationshipsCommand(),
			BetaGroupsDeleteCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// BetaGroupsListCommand returns the beta groups list subcommand.
func BetaGroupsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ExitOnError)

	appID := fs.String("app", "", "App Store Connect app ID (or APPSTORE_APP_ID env)")
	global := fs.Bool("global", false, "List beta groups across all apps (top-level endpoint)")
	internal := fs.Bool("internal", false, "Filter to internal groups only")
	external := fs.Bool("external", false, "Filter to external groups only")
	output := shared.BindOutputFlags(fs)
	limit := fs.Int("limit", 0, "Maximum results per page (1-200)")
	next := fs.String("next", "", "Fetch next page using a links.next URL")
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages (aggregate results)")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "appstore testflight beta-groups list [flags]",
		ShortHelp:  "List TestFlight beta groups for an app or globally.",
		LongHelp: `List TestFlight beta groups for an app or globally.

Examples:
  appstore testflight beta-groups list --app "APP_ID"
  appstore testflight beta-groups list --app "APP_ID" --internal
  appstore testflight beta-groups list --app "APP_ID" --external
  appstore testflight beta-groups list --app "APP_ID" --limit 10
  appstore testflight beta-groups list --app "APP_ID" --paginate
  appstore testflight beta-groups list --global
  appstore testflight beta-groups list --global --limit 50
  appstore testflight beta-groups list --global --internal`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *limit != 0 && (*limit < 1 || *limit > 200) {
				return fmt.Errorf("beta-groups list: --limit must be between 1 and 200")
			}
			if err := shared.ValidateNextURL(*next); err != nil {
				return fmt.Errorf("beta-groups list: %w", err)
			}

			resolvedAppID := shared.ResolveAppID(*appID)

			if *internal && *external {
				fmt.Fprintln(os.Stderr, "Error: --internal and --external are mutually exclusive")
				return flag.ErrHelp
			}

			// Reject --global + --app combination (check explicit flag, not resolved value)
			if *global && strings.TrimSpace(*appID) != "" {
				fmt.Fprintln(os.Stderr, "Error: --global and --app are mutually exclusive")
				return flag.ErrHelp
			}

			// Require one of --app or --global (unless --next is provided)
			if !*global && resolvedAppID == "" && strings.TrimSpace(*next) == "" {
				fmt.Fprintf(os.Stderr, "Error: --app or --global is required (or set APPSTORE_APP_ID)\n\n")
				return flag.ErrHelp
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("beta-groups list: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			var internalFilter *bool
			if *internal {
				v := true
				internalFilter = &v
			} else if *external {
				v := false
				internalFilter = &v
			}

			opts := []asc.BetaGroupsOption{
				asc.WithBetaGroupsLimit(*limit),
				asc.WithBetaGroupsNextURL(*next),
			}

			if *global {
				if internalFilter != nil {
					opts = append(opts, asc.WithBetaGroupsIsInternal(*internalFilter))
				}

				if *paginate {
					paginateOpts := append(opts, asc.WithBetaGroupsLimit(200))
					groups, err := shared.PaginateWithSpinner(requestCtx,
						func(ctx context.Context) (asc.PaginatedResponse, error) {
							return client.ListBetaGroups(ctx, paginateOpts...)
						},
						func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
							return client.ListBetaGroups(ctx, asc.WithBetaGroupsNextURL(nextURL))
						},
					)
					if err != nil {
						return fmt.Errorf("beta-groups list: %w", err)
					}

					return shared.PrintOutput(groups, *output.Output, *output.Pretty)
				}

				groups, err := client.ListBetaGroups(requestCtx, opts...)
				if err != nil {
					return fmt.Errorf("beta-groups list: failed to fetch: %w", err)
				}

				return shared.PrintOutput(groups, *output.Output, *output.Pretty)
			}

			// The app-scoped endpoint /v1/apps/{id}/betaGroups does not accept
			// filter[isInternalGroup], so we apply the filter client-side.
			if internalFilter != nil {
				var groups *asc.BetaGroupsResponse

				if *paginate {
					paginateOpts := append(opts, asc.WithBetaGroupsLimit(200))
					resp, err := shared.PaginateWithSpinner(requestCtx,
						func(ctx context.Context) (asc.PaginatedResponse, error) {
							return client.GetBetaGroups(ctx, resolvedAppID, paginateOpts...)
						},
						func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
							return client.GetBetaGroups(ctx, resolvedAppID, asc.WithBetaGroupsNextURL(nextURL))
						},
					)
					if err != nil {
						return fmt.Errorf("beta-groups list: %w", err)
					}
					var ok bool
					groups, ok = resp.(*asc.BetaGroupsResponse)
					if !ok {
						return fmt.Errorf("beta-groups list: unexpected response type %T", resp)
					}
				} else {
					// To apply the filter correctly, fetch all pages even without --paginate.
					paginateOpts := append(opts, asc.WithBetaGroupsLimit(200))
					firstPage, err := client.GetBetaGroups(requestCtx, resolvedAppID, paginateOpts...)
					if err != nil {
						return fmt.Errorf("beta-groups list: failed to fetch: %w", err)
					}
					resp, err := asc.PaginateAll(requestCtx, firstPage, func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
						return client.GetBetaGroups(ctx, resolvedAppID, asc.WithBetaGroupsNextURL(nextURL))
					})
					if err != nil {
						return fmt.Errorf("beta-groups list: %w", err)
					}
					var ok bool
					groups, ok = resp.(*asc.BetaGroupsResponse)
					if !ok {
						return fmt.Errorf("beta-groups list: unexpected response type %T", resp)
					}
				}

				filtered := *groups
				filtered.Data = make([]asc.Resource[asc.BetaGroupAttributes], 0, len(groups.Data))
				for _, g := range groups.Data {
					if g.Attributes.IsInternalGroup == *internalFilter {
						filtered.Data = append(filtered.Data, g)
					}
				}
				if *limit > 0 && len(filtered.Data) > *limit {
					filtered.Data = filtered.Data[:*limit]
				}

				return shared.PrintOutput(&filtered, *output.Output, *output.Pretty)
			}

			if *paginate {
				paginateOpts := append(opts, asc.WithBetaGroupsLimit(200))
				groups, err := shared.PaginateWithSpinner(requestCtx,
					func(ctx context.Context) (asc.PaginatedResponse, error) {
						return client.GetBetaGroups(ctx, resolvedAppID, paginateOpts...)
					},
					func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
						return client.GetBetaGroups(ctx, resolvedAppID, asc.WithBetaGroupsNextURL(nextURL))
					},
				)
				if err != nil {
					return fmt.Errorf("beta-groups list: %w", err)
				}

				return shared.PrintOutput(groups, *output.Output, *output.Pretty)
			}

			groups, err := client.GetBetaGroups(requestCtx, resolvedAppID, opts...)
			if err != nil {
				return fmt.Errorf("beta-groups list: failed to fetch: %w", err)
			}

			return shared.PrintOutput(groups, *output.Output, *output.Pretty)
		},
	}
}

// BetaGroupsCreateCommand returns the beta groups create subcommand.
func BetaGroupsCreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ExitOnError)

	appID := fs.String("app", "", "App Store Connect app ID (or APPSTORE_APP_ID env)")
	name := fs.String("name", "", "Beta group name")
	internal := fs.Bool("internal", false, "Create as internal group")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "appstore testflight beta-groups create [flags]",
		ShortHelp:  "Create a TestFlight beta group.",
		LongHelp: `Create a TestFlight beta group.

Examples:
  appstore testflight beta-groups create --app "APP_ID" --name "Beta Testers"
  appstore testflight beta-groups create --app "APP_ID" --name "Internal Testers" --internal`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			resolvedAppID := shared.ResolveAppID(*appID)
			if resolvedAppID == "" {
				fmt.Fprintf(os.Stderr, "Error: --app is required (or set APPSTORE_APP_ID)\n\n")
				return flag.ErrHelp
			}
			if strings.TrimSpace(*name) == "" {
				fmt.Fprintln(os.Stderr, "Error: --name is required")
				return flag.ErrHelp
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("beta-groups create: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			attrs := asc.BetaGroupAttributes{
				Name: strings.TrimSpace(*name),
			}
			if *internal {
				attrs.IsInternalGroup = true
			}

			group, err := client.CreateBetaGroupWithAttributes(requestCtx, resolvedAppID, attrs)
			if err != nil {
				return fmt.Errorf("beta-groups create: failed to create: %w", err)
			}

			return shared.PrintOutput(group, *output.Output, *output.Pretty)
		},
	}
}

// BetaGroupsGetCommand returns the beta groups get subcommand.
func BetaGroupsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ExitOnError)

	id := fs.String("id", "", "Beta group ID")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "appstore testflight beta-groups get [flags]",
		ShortHelp:  "Get a TestFlight beta group by ID.",
		LongHelp: `Get a TestFlight beta group by ID.

Examples:
  appstore testflight beta-groups get --id "GROUP_ID"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*id) == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return flag.ErrHelp
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("beta-groups get: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			group, err := client.GetBetaGroup(requestCtx, strings.TrimSpace(*id))
			if err != nil {
				return fmt.Errorf("beta-groups get: failed to fetch: %w", err)
			}

			return shared.PrintOutput(group, *output.Output, *output.Pretty)
		},
	}
}

// BetaGroupsUpdateCommand returns the beta groups update subcommand.
func BetaGroupsUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ExitOnError)

	id := fs.String("id", "", "Beta group ID")
	name := fs.String("name", "", "Beta group name")
	publicLinkEnabled := fs.Bool("public-link-enabled", false, "Enable public link")
	publicLinkLimitEnabled := fs.Bool("public-link-limit-enabled", false, "Enable public link limit")
	publicLinkLimit := fs.Int("public-link-limit", 0, "Public link limit (1-10000)")
	feedbackEnabled := fs.Bool("feedback-enabled", false, "Enable feedback")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "appstore testflight beta-groups update [flags]",
		ShortHelp:  "Update a TestFlight beta group.",
		LongHelp: `Update a TestFlight beta group.

Examples:
  appstore testflight beta-groups update --id "GROUP_ID" --name "New Name"
  appstore testflight beta-groups update --id "GROUP_ID" --public-link-enabled --public-link-limit 100
  appstore testflight beta-groups update --id "GROUP_ID" --feedback-enabled`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			trimmedID := strings.TrimSpace(*id)
			if trimmedID == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return flag.ErrHelp
			}

			visited := map[string]bool{}
			fs.Visit(func(f *flag.Flag) {
				visited[f.Name] = true
			})

			if visited["public-link-limit"] && (*publicLinkLimit < 1 || *publicLinkLimit > 10000) {
				fmt.Fprintln(os.Stderr, "Error: --public-link-limit must be between 1 and 10000")
				return flag.ErrHelp
			}

			hasUpdates := strings.TrimSpace(*name) != "" ||
				visited["public-link-enabled"] ||
				visited["public-link-limit-enabled"] ||
				visited["public-link-limit"] ||
				visited["feedback-enabled"]
			if !hasUpdates {
				fmt.Fprintln(os.Stderr, "Error: at least one update flag is required")
				return flag.ErrHelp
			}

			if visited["public-link-limit-enabled"] && *publicLinkLimitEnabled && !visited["public-link-limit"] {
				fmt.Fprintln(os.Stderr, "Error: --public-link-limit is required when enabling public link limit")
				return flag.ErrHelp
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("beta-groups update: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			var publicLinkEnabledAttr *bool
			var publicLinkLimitEnabledAttr *bool
			var feedbackEnabledAttr *bool

			if visited["public-link-enabled"] {
				publicLinkEnabledAttr = publicLinkEnabled
			}
			if visited["public-link-limit-enabled"] {
				publicLinkLimitEnabledAttr = publicLinkLimitEnabled
			}
			if visited["feedback-enabled"] {
				feedbackEnabledAttr = feedbackEnabled
			}

			req := asc.BetaGroupUpdateRequest{
				Data: asc.BetaGroupUpdateData{
					Type: asc.ResourceTypeBetaGroups,
					ID:   trimmedID,
					Attributes: &asc.BetaGroupUpdateAttributes{
						Name:                   strings.TrimSpace(*name),
						PublicLinkEnabled:      publicLinkEnabledAttr,
						PublicLinkLimitEnabled: publicLinkLimitEnabledAttr,
						PublicLinkLimit:        *publicLinkLimit,
						FeedbackEnabled:        feedbackEnabledAttr,
					},
				},
			}

			group, err := client.UpdateBetaGroup(requestCtx, trimmedID, req)
			if err != nil {
				return fmt.Errorf("beta-groups update: failed to update: %w", err)
			}

			return shared.PrintOutput(group, *output.Output, *output.Pretty)
		},
	}
}

// BetaGroupsDeleteCommand returns the beta groups delete subcommand.
func BetaGroupsDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)

	id := fs.String("id", "", "Beta group ID")
	confirm := fs.Bool("confirm", false, "Confirm deletion")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "appstore testflight beta-groups delete --id \"GROUP_ID\" --confirm",
		ShortHelp:  "Delete a TestFlight beta group.",
		LongHelp: `Delete a TestFlight beta group.

Examples:
  appstore testflight beta-groups delete --id "GROUP_ID" --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*id) == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return flag.ErrHelp
			}
			if !*confirm {
				fmt.Fprintln(os.Stderr, "Error: --confirm is required to delete")
				return flag.ErrHelp
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("beta-groups delete: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			if err := client.DeleteBetaGroup(requestCtx, strings.TrimSpace(*id)); err != nil {
				return fmt.Errorf("beta-groups delete: failed to delete: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Successfully deleted beta group %s\n", strings.TrimSpace(*id))
			return nil
		},
	}
}

// BetaGroupsAddTestersCommand returns the beta groups add-testers subcommand.
func BetaGroupsAddTestersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("add-testers", flag.ExitOnError)

	group := fs.String("group", "", "Beta group ID")
	tester := fs.String("tester", "", "Beta tester ID(s), comma-separated")

	return &ffcli.Command{
		Name:       "add-testers",
		ShortUsage: "appstore testflight beta-groups add-testers --group \"GROUP_ID\" --tester \"TESTER_ID[,TESTER_ID...]\"",
		ShortHelp:  "Add beta testers to a beta group.",
		LongHelp: `Add beta testers to a beta group.

Examples:
  appstore testflight beta-groups add-testers --group "GROUP_ID" --tester "TESTER_ID"
  appstore testflight beta-groups add-testers --group "GROUP_ID" --tester "TESTER_ID1,TESTER_ID2"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			groupID := strings.TrimSpace(*group)
			if groupID == "" {
				fmt.Fprintln(os.Stderr, "Error: --group is required")
				return flag.ErrHelp
			}

			testerIDs := shared.SplitCSV(*tester)
			if len(testerIDs) == 0 {
				fmt.Fprintln(os.Stderr, "Error: --tester is required")
				return flag.ErrHelp
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("beta-groups add-testers: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			if err := client.AddBetaTestersToGroup(requestCtx, groupID, testerIDs); err != nil {
				return fmt.Errorf("beta-groups add-testers: failed to add testers: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Successfully added %d tester(s) to group %s\n", len(testerIDs), groupID)
			return nil
		},
	}
}

// BetaGroupsRemoveTestersCommand returns the beta groups remove-testers subcommand.
func BetaGroupsRemoveTestersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("remove-testers", flag.ExitOnError)

	group := fs.String("group", "", "Beta group ID")
	tester := fs.String("tester", "", "Beta tester ID(s), comma-separated")
	confirm := fs.Bool("confirm", false, "Confirm removal")

	return &ffcli.Command{
		Name:       "remove-testers",
		ShortUsage: "appstore testflight beta-groups remove-testers --group \"GROUP_ID\" --tester \"TESTER_ID[,TESTER_ID...]\" --confirm",
		ShortHelp:  "Remove beta testers from a beta group.",
		LongHelp: `Remove beta testers from a beta group.

Examples:
  appstore testflight beta-groups remove-testers --group "GROUP_ID" --tester "TESTER_ID" --confirm
  appstore testflight beta-groups remove-testers --group "GROUP_ID" --tester "TESTER_ID1,TESTER_ID2" --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			groupID := strings.TrimSpace(*group)
			if groupID == "" {
				fmt.Fprintln(os.Stderr, "Error: --group is required")
				return flag.ErrHelp
			}

			testerIDs := shared.SplitCSV(*tester)
			if len(testerIDs) == 0 {
				fmt.Fprintln(os.Stderr, "Error: --tester is required")
				return flag.ErrHelp
			}
			if !*confirm {
				fmt.Fprintln(os.Stderr, "Error: --confirm is required")
				return flag.ErrHelp
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("beta-groups remove-testers: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			if err := client.RemoveBetaTestersFromGroup(requestCtx, groupID, testerIDs); err != nil {
				return fmt.Errorf("beta-groups remove-testers: failed to remove testers: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Successfully removed %d tester(s) from group %s\n", len(testerIDs), groupID)
			return nil
		},
	}
}
