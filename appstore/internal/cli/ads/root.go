package ads

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// AdsCommand returns the Apple Ads root command.
func AdsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("ads", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "ads",
		ShortUsage: "asc ads <subcommand> [flags]",
		ShortHelp:  "Manage Apple Ads API resources.",
		LongHelp: `Manage Apple Ads API resources.

Apple Ads credentials are separate from App Store Connect API credentials.

Examples:
  asc ads auth login --name "Ads" --client-id "SEARCHADS..." --team-id "SEARCHADS..." --key-id "KEY_ID" --private-key ./private-key.pem --ad-account "123456"
  asc ads auth discover --output json
  asc ads apps search --ad-account "123456" --query "Example"
  asc ads api request --method GET --path v1/me
  asc ads v5 campaigns list --org "123456" --limit 10`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: append([]*ffcli.Command{
			AuthCommand(),
			PlatformAPICommand(),
			V5Command(),
		}, platformEndpointCommands()...),
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// V5Command returns the deprecated Campaign Management API v5 command group.
func V5Command() *ffcli.Command {
	fs := flag.NewFlagSet("ads v5", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "v5",
		ShortUsage: "asc ads v5 <subcommand> [flags]",
		ShortHelp:  "DEPRECATED: Manage Campaign Management API v5 resources.",
		LongHelp: `DEPRECATED: Apple Ads Campaign Management API v5 retires on January 26, 2027.

The direct asc ads resource commands use Platform API v1. Use this namespace
only while migrating legacy Campaign Management API v5 automation.

Examples:
  asc ads v5 campaigns list --org "123456" --limit 10
  asc ads v5 api request --method GET --path v5/me`,
		FlagSet:     fs,
		UsageFunc:   shared.DefaultUsageFunc,
		Subcommands: append([]*ffcli.Command{APICommand()}, legacyEndpointCommands()...),
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
