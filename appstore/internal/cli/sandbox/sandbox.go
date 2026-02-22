package sandbox

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// SandboxCommand returns the sandbox testers command with subcommands.
func SandboxCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sandbox", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "sandbox",
		ShortUsage: "appstore sandbox <subcommand> [flags]",
		ShortHelp:  "Manage sandbox testers in App Store Connect.",
		LongHelp: `Manage sandbox testers for in-app purchase testing.

Examples:
  appstore sandbox list
  appstore sandbox list --email "tester@example.com"
  appstore sandbox get --id "SANDBOX_TESTER_ID"
  appstore sandbox update --id "SANDBOX_TESTER_ID" --territory "USA"
  appstore sandbox clear-history --id "SANDBOX_TESTER_ID" --confirm
`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			SandboxListCommand(),
			SandboxGetCommand(),
			SandboxUpdateCommand(),
			SandboxClearHistoryCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
