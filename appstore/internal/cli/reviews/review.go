package reviews

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// ReviewCommand returns the review parent command.
func ReviewCommand() *ffcli.Command {
	fs := flag.NewFlagSet("review", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "review",
		ShortUsage: "appstore review <subcommand> [flags]",
		ShortHelp:  "Manage App Store review details, attachments, and submissions.",
		LongHelp: `Manage App Store review details, attachments, submissions, and items.

Examples:
  appstore review details-get --id "DETAIL_ID"
  appstore review details-for-version --version-id "VERSION_ID"
  appstore review details-create --version-id "VERSION_ID" --contact-email "dev@example.com"
  appstore review details-update --id "DETAIL_ID" --notes "Updated review notes"
  appstore review attachments-list --review-detail "DETAIL_ID"
  appstore review submissions-list --app "123456789"
  appstore review submissions-create --app "123456789" --platform IOS
  appstore review submissions-submit --id "SUBMISSION_ID" --confirm
  appstore review submissions-update --id "SUBMISSION_ID" --canceled true
  appstore review submissions-items-ids --id "SUBMISSION_ID"
  appstore review items-get --id "ITEM_ID"
  appstore review items-add --submission "SUBMISSION_ID" --item-type appStoreVersions --item-id "VERSION_ID"
  appstore review items-update --id "ITEM_ID" --state READY_FOR_REVIEW`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ReviewDetailsGetCommand(),
			ReviewDetailsForVersionCommand(),
			ReviewDetailsCreateCommand(),
			ReviewDetailsUpdateCommand(),
			ReviewDetailsAttachmentsListCommand(),
			ReviewDetailsAttachmentsGetCommand(),
			ReviewDetailsAttachmentsUploadCommand(),
			ReviewDetailsAttachmentsDeleteCommand(),
			ReviewSubmissionsListCommand(),
			ReviewSubmissionsGetCommand(),
			ReviewSubmissionsCreateCommand(),
			ReviewSubmissionsSubmitCommand(),
			ReviewSubmissionsCancelCommand(),
			ReviewSubmissionsUpdateCommand(),
			ReviewSubmissionsItemsIDsCommand(),
			ReviewItemsGetCommand(),
			ReviewItemsListCommand(),
			ReviewItemsAddCommand(),
			ReviewItemsUpdateCommand(),
			ReviewItemsRemoveCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
