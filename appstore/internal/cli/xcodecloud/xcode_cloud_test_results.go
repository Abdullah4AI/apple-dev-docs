package xcodecloud

import "github.com/peterbourgon/ff/v3/ffcli"

// XcodeCloudTestResultsCommand returns the xcode-cloud test-results command with subcommands.
func XcodeCloudTestResultsCommand() *ffcli.Command {
	return newXcodeCloudActionResourceCommand(xcodeCloudTestResultsCommandConfig)
}
