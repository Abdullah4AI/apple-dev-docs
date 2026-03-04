package xcodecloud

import "github.com/peterbourgon/ff/v3/ffcli"

// XcodeCloudIssuesCommand returns the xcode-cloud issues command with subcommands.
func XcodeCloudIssuesCommand() *ffcli.Command {
	return newXcodeCloudActionResourceCommand(xcodeCloudIssuesCommandConfig)
}
