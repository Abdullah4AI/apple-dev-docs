package app_events

import (
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// SetClientFactory replaces the ASC client factory for tests.
// It returns a restore function to reset the previous handler.
func SetClientFactory(fn func() (*asc.Client, error)) func() {
	previous := appEventsClientFactory
	if fn == nil {
		appEventsClientFactory = shared.GetASCClient
	} else {
		appEventsClientFactory = fn
	}
	return func() {
		appEventsClientFactory = previous
	}
}
