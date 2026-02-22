package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/registry"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared/suggest"
)

var versionRequested bool

// RootCommand returns the root command
func RootCommand(version string) *ffcli.Command {
	versionRequested = false
	root := &ffcli.Command{
		Name:        "appstore",
		ShortUsage:  "appstore <subcommand> [flags]",
		ShortHelp:   "A fast, lightweight cli for App Store Connect. Built for developers and AI agents.",
		LongHelp:    "",
		FlagSet:     flag.NewFlagSet("appstore", flag.ExitOnError),
		UsageFunc:   RootUsageFunc,
		Subcommands: registry.Subcommands(version),
	}

	root.FlagSet.BoolVar(&versionRequested, "version", false, "Print version and exit")
	shared.BindRootFlags(root.FlagSet)

	rootSubcommandNames := make([]string, 0, len(root.Subcommands))
	for _, sub := range root.Subcommands {
		rootSubcommandNames = append(rootSubcommandNames, sub.Name)
	}

	root.Exec = func(ctx context.Context, args []string) error {
		if versionRequested {
			fmt.Fprintln(os.Stdout, version)
			return nil
		}
		if len(args) > 0 {
			unknown := shared.SanitizeTerminal(args[0])
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", unknown)
			if suggestions := suggest.Commands(args[0], rootSubcommandNames); len(suggestions) > 0 {
				for i, suggestion := range suggestions {
					suggestions[i] = shared.SanitizeTerminal(suggestion)
				}
				fmt.Fprintf(os.Stderr, "Did you mean: %s\n\n", strings.Join(suggestions, ", "))
			}
		}
		return flag.ErrHelp
	}

	return root
}
