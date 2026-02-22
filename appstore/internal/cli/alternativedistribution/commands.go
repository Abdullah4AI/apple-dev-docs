package alternativedistribution

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the alternative distribution command group.
func Command() *ffcli.Command {
	return AlternativeDistributionCommand()
}
