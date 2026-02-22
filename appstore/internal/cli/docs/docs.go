package docs

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// DocsCommand returns the docs command group.
func DocsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("docs", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "docs",
		ShortUsage: "appstore docs <subcommand> [flags]",
		ShortHelp:  "Access embedded documentation guides and reference helpers.",
		LongHelp: `Access embedded documentation guides and reference helpers.

Examples:
  appstore docs list
  appstore docs show workflows
  appstore docs init
  appstore docs init --path ./APPSTORE.md
  appstore docs init --force --link=false`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			DocsListCommand(),
			DocsShowCommand(),
			DocsInitCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", args[0])
			return flag.ErrHelp
		},
	}
}
