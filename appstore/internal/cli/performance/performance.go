package performance

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// PerformanceCommand returns the performance command group.
func PerformanceCommand() *ffcli.Command {
	fs := flag.NewFlagSet("performance", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "performance",
		ShortUsage: "appstore performance <subcommand> [flags]",
		ShortHelp:  "Access performance metrics and diagnostic logs.",
		LongHelp: `Access performance metrics and diagnostic logs.

Examples:
  appstore performance metrics list --app "APP_ID"
  appstore performance metrics get --build "BUILD_ID"
  appstore performance diagnostics list --build "BUILD_ID"
  appstore performance diagnostics get --id "SIGNATURE_ID"
  appstore performance download --build "BUILD_ID" --output ./metrics.json`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			PerformanceMetricsCommand(),
			PerformanceDiagnosticsCommand(),
			PerformanceDownloadCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
