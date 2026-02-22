package cmdtest

import (
	"github.com/peterbourgon/ff/v3/ffcli"

	cmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func RootCommand(version string) *ffcli.Command {
	return cmd.RootCommand(version)
}

type ReportedError = shared.ReportedError
