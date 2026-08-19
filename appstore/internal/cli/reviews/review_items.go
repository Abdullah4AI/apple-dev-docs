package reviews

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

var reviewItemsClientFactory = shared.GetASCClient

// ReviewItemsCommand returns the nested review items command group.
func ReviewItemsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("items", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "items",
		ShortUsage: "asc review items <subcommand> [flags]",
		ShortHelp:  "Manage review submission items.",
		LongHelp: `Manage review submission items.

Examples:
  asc review items list --submission "SUBMISSION_ID"
  asc review items add --submission "SUBMISSION_ID" --item-type appStoreVersions --item-id "VERSION_ID"
  asc review items add --submission "SUBMISSION_ID" --item-type inAppPurchaseVersions --item-id "IAP_VERSION_ID"
  asc review items add --submission "SUBMISSION_ID" --item-type subscriptionVersions --item-id "SUBSCRIPTION_VERSION_ID"
  asc review items add --submission "SUBMISSION_ID" --item-type subscriptionGroupVersions --item-id "GROUP_VERSION_ID"
  asc review items update --id "ITEM_ID" --resolved true
  asc review items remove --id "ITEM_ID" --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			reviewItemsListCommand("list", "review items list", `asc review items list [flags]`, `asc review items list --submission "SUBMISSION_ID"
  asc review items list --submission "SUBMISSION_ID" --paginate`),
			reviewItemsAddCommand("add", "review items add", `asc review items add [flags]`, `asc review items add --submission "SUBMISSION_ID" --item-type appStoreVersions --item-id "VERSION_ID"
  asc review items add --submission "SUBMISSION_ID" --item-type inAppPurchaseVersions --item-id "IAP_VERSION_ID"
  asc review items add --submission "SUBMISSION_ID" --item-type subscriptionVersions --item-id "SUBSCRIPTION_VERSION_ID"
  asc review items add --submission "SUBMISSION_ID" --item-type subscriptionGroupVersions --item-id "GROUP_VERSION_ID"
  asc review items add --submission "SUBMISSION_ID" --item-type gameCenterChallengeVersions --item-id "VERSION_ID"`),
			reviewItemsUpdateCommand("update", "review items update", `asc review items update --id "ITEM_ID" [flags]`, `asc review items update --id "ITEM_ID" --resolved true
  asc review items update --id "ITEM_ID" --clear-removed`),
			reviewItemsRemoveCommand("remove", "review items remove", `asc review items remove [flags]`, `asc review items remove --id "ITEM_ID" --confirm`),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				subcommand := strings.TrimSpace(args[0])
				if subcommand == "view" {
					return removedReviewItemDetailUsageError("asc review items view")
				}
				return shared.WithDiagnostic(shared.UsageErrorf("unexpected argument(s): %s", shared.SanitizeTerminal(subcommand)), shared.DiagnosticInvalidInput, "")
			}
			return flag.ErrHelp
		},
	}
}

func removedReviewItemDetailUsageError(command string) error {
	return shared.WithDiagnostic(shared.UsageErrorf(
		"`%s` was removed in 4.0.0; use `asc review items list --submission \"SUBMISSION_ID\"` instead",
		command,
	), shared.DiagnosticInvalidInput, "")
}

// ReviewItemsListCommand returns the review items list subcommand.
func ReviewItemsListCommand() *ffcli.Command {
	return reviewItemsListCommand("items-list", "review items-list", `asc review items-list [flags]`, `asc review items-list --submission "SUBMISSION_ID"
  asc review items-list --submission "SUBMISSION_ID" --paginate`)
}

func reviewItemsListCommand(name, errorPrefix, shortUsage, examples string) *ffcli.Command {
	fs := flag.NewFlagSet(name, flag.ExitOnError)

	submissionID := fs.String("submission", "", "Review submission ID (required)")
	limit := fs.Int("limit", 0, "Maximum results per page (1-200)")
	next := fs.String("next", "", "Next page URL from a previous response")
	fields := fs.String("fields", "", "Review item fields: "+strings.Join(reviewSubmissionItemFields, ", "))
	include := fs.String("include", "", "Include relationships: "+strings.Join(reviewSubmissionItemIncludes, ", "))
	iapVersionFields := fs.String("iap-version-fields", "", "In-app purchase version fields: "+strings.Join(reviewSubmissionItemIAPVersionFields, ", "))
	subscriptionVersionFields := fs.String("subscription-version-fields", "", "Subscription version fields: "+strings.Join(reviewSubmissionItemSubscriptionVersionFields, ", "))
	subscriptionGroupVersionFields := fs.String("subscription-group-version-fields", "", "Subscription group version fields: "+strings.Join(reviewSubmissionItemSubscriptionGroupVersionFields, ", "))
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages (aggregate results)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       name,
		ShortUsage: shortUsage,
		ShortHelp:  "List items in a review submission.",
		LongHelp: `List items in a review submission.

Examples:
  ` + examples,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("%s: %w", errorPrefix, shared.WithDiagnostic(shared.UsageError("unexpected positional arguments"), shared.DiagnosticInvalidInput, ""))
			}
			if err := shared.ValidateNextURL(*next); err != nil {
				return shared.WithDiagnostic(shared.NewValidationError(fmt.Errorf("%s: %w", errorPrefix, err)), shared.DiagnosticInvalidInput, "--next")
			}
			if err := rejectReviewNextFlagConflicts(
				fs, *next, errorPrefix,
				"submission", "limit", "fields", "include", "iap-version-fields",
				"subscription-version-fields", "subscription-group-version-fields",
			); err != nil {
				return err
			}
			opts, err := reviewItemsListOptions(*limit, *next, *fields, *include, *iapVersionFields, *subscriptionVersionFields, *subscriptionGroupVersionFields)
			if err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}
			if strings.TrimSpace(*submissionID) == "" && strings.TrimSpace(*next) == "" {
				fmt.Fprintln(os.Stderr, "Error: --submission is required")
				return shared.MissingRequiredUsageError("--submission")
			}

			client, err := reviewItemsClientFactory()
			if err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			if *paginate {
				paginateOpts := append(opts, asc.WithReviewSubmissionItemsLimit(200))
				resp, err := shared.PaginateWithSpinner(
					requestCtx,
					func(ctx context.Context) (asc.PaginatedResponse, error) {
						return client.GetReviewSubmissionItems(ctx, strings.TrimSpace(*submissionID), paginateOpts...)
					},
					func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
						return client.GetReviewSubmissionItems(ctx, strings.TrimSpace(*submissionID), asc.WithReviewSubmissionItemsNextURL(nextURL))
					},
				)
				if err != nil {
					return fmt.Errorf("%s: %w", errorPrefix, err)
				}

				return shared.PrintOutput(resp, *output.Output, *output.Pretty)
			}

			resp, err := client.GetReviewSubmissionItems(requestCtx, strings.TrimSpace(*submissionID), opts...)
			if err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}

			return shared.PrintOutput(resp, *output.Output, *output.Pretty)
		},
	}
}

func rejectReviewNextFlagConflicts(fs *flag.FlagSet, next, command string, names ...string) error {
	if strings.TrimSpace(next) == "" {
		return nil
	}
	provided := make(map[string]struct{}, len(names))
	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = struct{}{}
	})
	for _, name := range names {
		if _, ok := provided[name]; ok {
			return shared.WithDiagnostic(shared.UsageErrorf("%s: --next cannot be combined with --%s", command, name), shared.DiagnosticConflictingInput, "")
		}
	}
	return nil
}

var reviewSubmissionItemFields = []string{
	"state", "appStoreVersion", "appCustomProductPageVersion", "appStoreVersionExperiment",
	"appStoreVersionExperimentV2", "appEvent", "backgroundAssetVersion", "gameCenterAchievementVersion",
	"gameCenterActivityVersion", "gameCenterChallengeVersion", "gameCenterLeaderboardSetVersion",
	"gameCenterLeaderboardVersion", "inAppPurchaseVersion", "subscriptionVersion", "subscriptionGroupVersion",
}

var reviewSubmissionItemIncludes = reviewSubmissionItemFields[1:]

var (
	reviewSubmissionItemIAPVersionFields               = []string{"version", "state", "inAppPurchase", "image", "images", "localizations"}
	reviewSubmissionItemSubscriptionVersionFields      = []string{"version", "state", "subscription", "image", "images", "localizations"}
	reviewSubmissionItemSubscriptionGroupVersionFields = []string{"version", "state", "subscriptionGroup", "localizations"}
)

func reviewItemsListOptions(limit int, next, fields, include, iapVersionFields, subscriptionVersionFields, subscriptionGroupVersionFields string) ([]asc.ReviewSubmissionItemsOption, error) {
	if limit != 0 && (limit < 1 || limit > 200) {
		return nil, shared.WithDiagnostic(shared.UsageError("--limit must be between 1 and 200"), shared.DiagnosticInvalidInput, "--limit")
	}
	if err := shared.ValidateNextURL(next); err != nil {
		return nil, shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--next")
	}
	if strings.TrimSpace(next) != "" && (limit != 0 || strings.TrimSpace(fields) != "" || strings.TrimSpace(include) != "" ||
		strings.TrimSpace(iapVersionFields) != "" || strings.TrimSpace(subscriptionVersionFields) != "" || strings.TrimSpace(subscriptionGroupVersionFields) != "") {
		return nil, shared.WithDiagnostic(shared.UsageError("--next cannot be combined with --limit, --fields, --include, or version sparse-field flags"), shared.DiagnosticConflictingInput, "")
	}

	itemFields, err := shared.NormalizeSelection(fields, reviewSubmissionItemFields, "--fields")
	if err != nil {
		return nil, shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--fields")
	}
	includes, err := shared.NormalizeSelection(include, reviewSubmissionItemIncludes, "--include")
	if err != nil {
		return nil, shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--include")
	}
	iapFields, err := shared.NormalizeSelection(iapVersionFields, reviewSubmissionItemIAPVersionFields, "--iap-version-fields")
	if err != nil {
		return nil, shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--iap-version-fields")
	}
	subscriptionFields, err := shared.NormalizeSelection(subscriptionVersionFields, reviewSubmissionItemSubscriptionVersionFields, "--subscription-version-fields")
	if err != nil {
		return nil, shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--subscription-version-fields")
	}
	groupFields, err := shared.NormalizeSelection(subscriptionGroupVersionFields, reviewSubmissionItemSubscriptionGroupVersionFields, "--subscription-group-version-fields")
	if err != nil {
		return nil, shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--subscription-group-version-fields")
	}
	addVersionRelationship := func(versionFields []string, relationship string) {
		if len(versionFields) == 0 {
			return
		}
		if !slices.Contains(includes, relationship) {
			includes = append(includes, relationship)
		}
		if len(itemFields) != 0 && !slices.Contains(itemFields, relationship) {
			itemFields = append(itemFields, relationship)
		}
	}
	addVersionRelationship(iapFields, "inAppPurchaseVersion")
	addVersionRelationship(subscriptionFields, "subscriptionVersion")
	addVersionRelationship(groupFields, "subscriptionGroupVersion")

	return []asc.ReviewSubmissionItemsOption{
		asc.WithReviewSubmissionItemsLimit(limit),
		asc.WithReviewSubmissionItemsNextURL(next),
		asc.WithReviewSubmissionItemsFields(itemFields),
		asc.WithReviewSubmissionItemsInclude(includes),
		asc.WithReviewSubmissionItemsInAppPurchaseVersionFields(iapFields),
		asc.WithReviewSubmissionItemsSubscriptionVersionFields(subscriptionFields),
		asc.WithReviewSubmissionItemsSubscriptionGroupVersionFields(groupFields),
	}, nil
}

// ReviewItemsAddCommand returns the review items add subcommand.
func ReviewItemsAddCommand() *ffcli.Command {
	return reviewItemsAddCommand("items-add", "review items-add", `asc review items-add [flags]`, `asc review items-add --submission "SUBMISSION_ID" --item-type appStoreVersions --item-id "VERSION_ID"
  asc review items-add --submission "SUBMISSION_ID" --item-type inAppPurchaseVersions --item-id "IAP_VERSION_ID"
  asc review items-add --submission "SUBMISSION_ID" --item-type subscriptionVersions --item-id "SUBSCRIPTION_VERSION_ID"
  asc review items-add --submission "SUBMISSION_ID" --item-type subscriptionGroupVersions --item-id "GROUP_VERSION_ID"
  asc review items-add --submission "SUBMISSION_ID" --item-type gameCenterChallengeVersions --item-id "VERSION_ID"`)
}

func reviewItemsAddCommand(name, errorPrefix, shortUsage, examples string) *ffcli.Command {
	fs := flag.NewFlagSet(name, flag.ExitOnError)

	submissionID := fs.String("submission", "", "Review submission ID (required)")
	itemTypeValues := strings.Join(reviewSubmissionItemTypeList(), ", ")
	itemType := fs.String("item-type", "", fmt.Sprintf("Item type: %s (required)", itemTypeValues))
	itemID := fs.String("item-id", "", "Item ID (required)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       name,
		ShortUsage: shortUsage,
		ShortHelp:  "Add an item to a review submission.",
		LongHelp: `Add an item to a review submission.

Examples:
  ` + examples,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("%s: %w", errorPrefix, shared.WithDiagnostic(shared.UsageError("unexpected positional arguments"), shared.DiagnosticInvalidInput, ""))
			}
			if strings.TrimSpace(*submissionID) == "" {
				fmt.Fprintln(os.Stderr, "Error: --submission is required")
				return shared.MissingRequiredUsageError("--submission")
			}
			if strings.TrimSpace(*itemType) == "" {
				fmt.Fprintln(os.Stderr, "Error: --item-type is required")
				return shared.MissingRequiredUsageError("--item-type")
			}
			if strings.TrimSpace(*itemID) == "" {
				fmt.Fprintln(os.Stderr, "Error: --item-id is required")
				return shared.MissingRequiredUsageError("--item-id")
			}

			normalizedType, err := normalizeReviewSubmissionItemType(*itemType)
			if err != nil {
				return shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--item-type")
			}

			client, err := reviewItemsClientFactory()
			if err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			resp, err := client.CreateReviewSubmissionItem(requestCtx, strings.TrimSpace(*submissionID), normalizedType, strings.TrimSpace(*itemID))
			if err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}

			return shared.PrintOutput(resp, *output.Output, *output.Pretty)
		},
	}
}

// ReviewItemsUpdateCommand returns the review items update subcommand.
func ReviewItemsUpdateCommand() *ffcli.Command {
	return reviewItemsUpdateCommand("items-update", "review items-update", `asc review items-update --id "ITEM_ID" [flags]`, `asc review items-update --id "ITEM_ID" --resolved true
  asc review items-update --id "ITEM_ID" --clear-removed`)
}

func reviewItemsUpdateCommand(name, errorPrefix, shortUsage, examples string) *ffcli.Command {
	fs := flag.NewFlagSet(name, flag.ExitOnError)

	itemID := fs.String("id", "", "Review submission item ID (required)")
	resolved := fs.String("resolved", "", "Whether the item is resolved: true or false")
	removed := fs.String("removed", "", "Whether the item is removed: true or false")
	clearResolved := fs.Bool("clear-resolved", false, "Set resolved to JSON null")
	clearRemoved := fs.Bool("clear-removed", false, "Set removed to JSON null")
	confirm := fs.Bool("confirm", false, "Confirm removal when --removed=true")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       name,
		ShortUsage: shortUsage,
		ShortHelp:  "Update a review submission item's resolved or removed status.",
		LongHelp: `Update a review submission item.

Use --confirm when setting --removed=true.

Examples:
  ` + examples,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("%s: %w", errorPrefix, shared.WithDiagnostic(shared.UsageError("unexpected positional arguments"), shared.DiagnosticInvalidInput, ""))
			}
			trimmedID := strings.TrimSpace(*itemID)
			if trimmedID == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return shared.MissingRequiredUsageError("--id")
			}
			resolvedProvided := reviewFlagWasProvided(fs, "resolved")
			removedProvided := reviewFlagWasProvided(fs, "removed")
			if resolvedProvided && *clearResolved {
				return fmt.Errorf("%s: %w", errorPrefix, shared.WithDiagnostic(shared.UsageError("--resolved cannot be combined with --clear-resolved"), shared.DiagnosticConflictingInput, ""))
			}
			if removedProvided && *clearRemoved {
				return fmt.Errorf("%s: %w", errorPrefix, shared.WithDiagnostic(shared.UsageError("--removed cannot be combined with --clear-removed"), shared.DiagnosticConflictingInput, ""))
			}
			if !resolvedProvided && !removedProvided && !*clearResolved && !*clearRemoved {
				return fmt.Errorf("%s: %w", errorPrefix, shared.WithDiagnostic(shared.UsageError("at least one of --resolved, --removed, --clear-resolved, or --clear-removed is required"), shared.DiagnosticRequiredInputMissing, ""))
			}

			attrs := asc.ReviewSubmissionItemUpdateAttributes{}
			if resolvedProvided {
				value, err := parseReviewSubmissionItemBool(*resolved, "--resolved")
				if err != nil {
					return fmt.Errorf("%s: %w", errorPrefix, err)
				}
				attrs.Resolved = &asc.NullableBool{Value: &value}
			} else if *clearResolved {
				attrs.Resolved = &asc.NullableBool{}
			}
			if removedProvided {
				value, err := parseReviewSubmissionItemBool(*removed, "--removed")
				if err != nil {
					return fmt.Errorf("%s: %w", errorPrefix, err)
				}
				if value && !*confirm {
					return fmt.Errorf("%s: %w", errorPrefix, shared.WithDiagnostic(shared.UsageError("--confirm is required when --removed=true"), shared.DiagnosticRequiredInputMissing, "--confirm"))
				}
				attrs.Removed = &asc.NullableBool{Value: &value}
			} else if *clearRemoved {
				attrs.Removed = &asc.NullableBool{}
			}

			client, err := reviewItemsClientFactory()
			if err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			resp, err := client.UpdateReviewSubmissionItem(requestCtx, trimmedID, attrs)
			if err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}

			return shared.PrintOutput(resp, *output.Output, *output.Pretty)
		},
	}
}

func reviewFlagWasProvided(fs *flag.FlagSet, names ...string) bool {
	wanted := make(map[string]struct{}, len(names))
	for _, name := range names {
		wanted[name] = struct{}{}
	}
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if _, ok := wanted[f.Name]; ok {
			provided = true
		}
	})
	return provided
}

func parseReviewSubmissionItemBool(value, name string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, shared.WithDiagnostic(shared.UsageErrorf("%s must be true or false", name), shared.DiagnosticInvalidInput, name)
	}
}

// ReviewItemsRemoveCommand returns the review items remove subcommand.
func ReviewItemsRemoveCommand() *ffcli.Command {
	return reviewItemsRemoveCommand("items-remove", "review items-remove", `asc review items-remove [flags]`, `asc review items-remove --id "ITEM_ID" --confirm`)
}

func reviewItemsRemoveCommand(name, errorPrefix, shortUsage, examples string) *ffcli.Command {
	fs := flag.NewFlagSet(name, flag.ExitOnError)

	itemID := fs.String("id", "", "Review submission item ID (required)")
	confirm := fs.Bool("confirm", false, "Confirm removal (required)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       name,
		ShortUsage: shortUsage,
		ShortHelp:  "Remove an item from a review submission.",
		LongHelp: `Remove an item from a review submission.

Examples:
  ` + examples,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("%s: %w", errorPrefix, shared.WithDiagnostic(shared.UsageError("unexpected positional arguments"), shared.DiagnosticInvalidInput, ""))
			}
			if !*confirm {
				fmt.Fprintln(os.Stderr, "Error: --confirm is required to remove")
				return shared.MissingRequiredUsageError("--confirm")
			}
			if strings.TrimSpace(*itemID) == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return shared.MissingRequiredUsageError("--id")
			}

			client, err := reviewItemsClientFactory()
			if err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			if err := client.DeleteReviewSubmissionItem(requestCtx, strings.TrimSpace(*itemID)); err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}

			result := &asc.ReviewSubmissionItemDeleteResult{
				ID:      strings.TrimSpace(*itemID),
				Deleted: true,
			}

			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}

func normalizeReviewSubmissionItemType(value string) (asc.ReviewSubmissionItemType, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("--item-type is required")
	}
	if guidance, ok := removedReviewSubmissionItemTypeGuidance(trimmed); ok {
		return "", errors.New(guidance)
	}
	if itemType, ok := asc.ParseReviewSubmissionItemType(value); ok {
		return itemType, nil
	}
	return "", fmt.Errorf("--item-type must be one of: %s", strings.Join(reviewSubmissionItemTypeList(), ", "))
}

// Item types App Store Connect stopped accepting as review submission items.
// They are rejected with targeted migration guidance instead of the generic
// supported-value list.
const (
	removedItemTypeCustomProductPages   = "appCustomProductPages"
	removedItemTypeExperimentTreatments = "appStoreVersionExperimentTreatments"
	removedItemTypeExperimentV2Alias    = "appStoreVersionExperimentV2"
)

func removedReviewSubmissionItemTypeGuidance(value string) (string, bool) {
	switch value {
	case removedItemTypeExperimentV2Alias:
		return fmt.Sprintf(
			"--item-type %s was removed in 4.0.0; use --item-type %s",
			removedItemTypeExperimentV2Alias, asc.ReviewSubmissionItemTypeAppStoreVersionExperimentV2,
		), true
	case removedItemTypeExperimentTreatments:
		return fmt.Sprintf(
			"--item-type %s is deprecated and no longer supported by App Store Connect; experiment treatments cannot be added as review submission items",
			removedItemTypeExperimentTreatments,
		), true
	case removedItemTypeCustomProductPages:
		return fmt.Sprintf(
			"--item-type %s is deprecated and no longer supported by App Store Connect; pass an app custom product page version ID with --item-type %s",
			removedItemTypeCustomProductPages, asc.ReviewSubmissionItemTypeAppCustomProductPageVersion,
		), true
	}
	return "", false
}

func reviewSubmissionItemTypeList() []string {
	return asc.ReviewSubmissionItemTypeNames()
}
