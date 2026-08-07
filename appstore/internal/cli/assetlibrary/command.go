package assetlibrary

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// Command returns the internal Asset Library command group.
//
// The group intentionally remains unregistered until Apple publishes the
// corresponding App Store Connect API contract.
func Command() *ffcli.Command {
	fs := flag.NewFlagSet("asset-library", flag.ExitOnError)

	return &ffcli.Command{
		Name:      "asset-library",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}
}
