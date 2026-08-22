package cmdtest

import (
	"errors"
	"flag"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"

	cmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/auth"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// isUsageClassError reports whether err maps to usage exit code 2. Commands
// reach that classification either by wrapping flag.ErrHelp, which makes ffcli
// render the full help page, or by returning a concise reported usage error
// whose message already stands on its own.
func isUsageClassError(err error) bool {
	return errors.Is(err, flag.ErrHelp) || shared.IsReportedUsageError(err)
}

func resetCmdtestState() {
	asc.ResetConfigCacheForTest()
	auth.ResetInvalidBypassKeychainWarningsForTest()
	shared.ResetDefaultOutputFormat()
	shared.ResetTierCacheForTest()
}

func setCmdtestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func RootCommand(version string) *ffcli.Command {
	resetCmdtestState()
	return cmd.RootCommand(version)
}

type ReportedError = shared.ReportedError
