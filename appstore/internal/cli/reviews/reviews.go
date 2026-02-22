package reviews

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

// ReviewsCommand returns the reviews command with subcommands.
func ReviewsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("reviews", flag.ExitOnError)

	appID := fs.String("app", "", "App Store Connect app ID (or APPSTORE_APP_ID env)")
	output := shared.BindOutputFlags(fs)
	stars := fs.Int("stars", 0, "Filter by star rating (1-5)")
	territory := fs.String("territory", "", "Filter by territory (e.g., US, GBR)")
	sort := fs.String("sort", "", "Sort by rating, -rating, createdDate, or -createdDate")
	limit := fs.Int("limit", 0, "Maximum results per page (1-200)")
	next := fs.String("next", "", "Fetch next page using a links.next URL")
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages (aggregate results)")

	return &ffcli.Command{
		Name:       "reviews",
		ShortUsage: "appstore reviews [flags] | appstore reviews <subcommand> [flags]",
		ShortHelp:  "List and manage App Store customer reviews.",
		LongHelp: `List and manage App Store customer reviews.

This command fetches customer reviews from the App Store,
helping you understand user feedback and sentiment.

When invoked with --app, lists reviews. Subcommands allow responding to reviews.

Examples:
  appstore reviews --app "123456789"
  appstore reviews --app "123456789" --stars 1 --territory US
  appstore reviews --app "123456789" --sort -createdDate --limit 5
  appstore reviews --next "<links.next>"
  appstore reviews --app "123456789" --paginate
  appstore reviews get --id "REVIEW_ID"
  appstore reviews ratings --app "123456789"
  appstore reviews ratings --app "123456789" --all
  appstore reviews summarizations --app "123456789" --platform IOS --territory US
  appstore reviews respond --review-id "REVIEW_ID" --response "Thanks!"
  appstore reviews response get --id "RESPONSE_ID"
  appstore reviews response delete --id "RESPONSE_ID" --confirm
  appstore reviews response for-review --review-id "REVIEW_ID"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ReviewsListCommand(),
			ReviewsGetCommand(),
			ReviewsRatingsCommand(),
			ReviewsSummarizationsCommand(),
			ReviewsRespondCommand(),
			ReviewsResponseCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			// If no flags are set and no args, show help
			resolvedAppID := shared.ResolveAppID(*appID)
			if resolvedAppID == "" && strings.TrimSpace(*next) == "" {
				fmt.Fprintf(os.Stderr, "Error: --app is required (or set APPSTORE_APP_ID)\n\n")
				return flag.ErrHelp
			}

			// Execute the list functionality directly
			return executeReviewsList(ctx, resolvedAppID, *output.Output, *output.Pretty, *stars, *territory, *sort, *limit, *next, *paginate)
		},
	}
}

// ReviewsListCommand returns the reviews list subcommand.
func ReviewsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ExitOnError)

	appID := fs.String("app", "", "App Store Connect app ID (or APPSTORE_APP_ID env)")
	output := shared.BindOutputFlags(fs)
	stars := fs.Int("stars", 0, "Filter by star rating (1-5)")
	territory := fs.String("territory", "", "Filter by territory (e.g., US, GBR)")
	sort := fs.String("sort", "", "Sort by rating, -rating, createdDate, or -createdDate")
	limit := fs.Int("limit", 0, "Maximum results per page (1-200)")
	next := fs.String("next", "", "Fetch next page using a links.next URL")
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages (aggregate results)")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "appstore reviews list [flags]",
		ShortHelp:  "List App Store customer reviews.",
		LongHelp: `List App Store customer reviews.

Examples:
  appstore reviews list --app "123456789"
  appstore reviews list --app "123456789" --stars 5
  appstore reviews list --app "123456789" --territory US --sort -createdDate
  appstore reviews list --next "<links.next>"
  appstore reviews list --app "123456789" --paginate`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			resolvedAppID := shared.ResolveAppID(*appID)
			if resolvedAppID == "" && strings.TrimSpace(*next) == "" {
				fmt.Fprintf(os.Stderr, "Error: --app is required (or set APPSTORE_APP_ID)\n\n")
				return flag.ErrHelp
			}

			return executeReviewsList(ctx, resolvedAppID, *output.Output, *output.Pretty, *stars, *territory, *sort, *limit, *next, *paginate)
		},
	}
}

func executeReviewsList(ctx context.Context, appID, output string, pretty bool, stars int, territory, sort string, limit int, next string, paginate bool) error {
	if limit != 0 && (limit < 1 || limit > 200) {
		return fmt.Errorf("reviews: --limit must be between 1 and 200")
	}
	if stars != 0 && (stars < 1 || stars > 5) {
		return fmt.Errorf("reviews: --stars must be between 1 and 5")
	}
	if err := shared.ValidateNextURL(next); err != nil {
		return fmt.Errorf("reviews: %w", err)
	}
	if err := shared.ValidateSort(sort, "rating", "-rating", "createdDate", "-createdDate"); err != nil {
		return fmt.Errorf("reviews: %w", err)
	}

	client, err := shared.GetASCClient()
	if err != nil {
		return fmt.Errorf("reviews: %w", err)
	}

	requestCtx, cancel := shared.ContextWithTimeout(ctx)
	defer cancel()

	opts := []asc.ReviewOption{
		asc.WithRating(stars),
		asc.WithTerritory(territory),
		asc.WithLimit(limit),
		asc.WithNextURL(next),
	}
	if strings.TrimSpace(sort) != "" {
		opts = append(opts, asc.WithReviewSort(sort))
	}

	if paginate {
		paginateOpts := append(opts, asc.WithLimit(200))
		reviews, err := shared.PaginateWithSpinner(requestCtx,
			func(ctx context.Context) (asc.PaginatedResponse, error) {
				return client.GetReviews(ctx, appID, paginateOpts...)
			},
			func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
				return client.GetReviews(ctx, appID, asc.WithNextURL(nextURL))
			},
		)
		if err != nil {
			return fmt.Errorf("reviews: %w", err)
		}

		return shared.PrintOutput(reviews, output, pretty)
	}

	reviews, err := client.GetReviews(requestCtx, appID, opts...)
	if err != nil {
		return fmt.Errorf("reviews: failed to fetch: %w", err)
	}

	return shared.PrintOutput(reviews, output, pretty)
}
