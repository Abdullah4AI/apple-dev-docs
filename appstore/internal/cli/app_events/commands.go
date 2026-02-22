package app_events

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the app-events command group.
func Command() *ffcli.Command {
	return AppEventsCommand()
}
