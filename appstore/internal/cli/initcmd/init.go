package initcmd

import (
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/docs"
)

// InitCommand returns the root init command.
func InitCommand() *ffcli.Command {
	return docs.NewInitReferenceCommand(
		"init",
		"init",
		"appstore init [flags]",
		"Initialize appstore helper docs in the current repo.",
		`Initialize appstore helper docs in the current repo.

Examples:
  appstore init
  appstore init --path ./APPSTORE.md
  appstore init --force --link=false`,
		"init",
	)
}
