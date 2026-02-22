package analytics

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// AnalyticsCommand returns the analytics command with subcommands.
func AnalyticsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("analytics", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "analytics",
		ShortUsage: "appstore analytics <subcommand> [flags]",
		ShortHelp:  "Request and download analytics and sales reports.",
		LongHelp: `Request and download analytics and sales reports.

Examples:
  appstore analytics sales --vendor "12345678" --type SALES --subtype SUMMARY --frequency DAILY --date "2024-01-20"
  appstore analytics request --app "APP_ID" --access-type ONGOING
  appstore analytics requests --app "APP_ID"
  appstore analytics get --request-id "REQUEST_ID"
  appstore analytics reports get --report-id "REPORT_ID"
  appstore analytics instances relationships --instance-id "INSTANCE_ID"
  appstore analytics download --request-id "REQUEST_ID" --instance-id "INSTANCE_ID"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AnalyticsSalesCommand(),
			AnalyticsRequestCommand(),
			AnalyticsRequestsCommand(),
			AnalyticsGetCommand(),
			AnalyticsReportsCommand(),
			AnalyticsInstancesCommand(),
			AnalyticsSegmentsCommand(),
			AnalyticsDownloadCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
