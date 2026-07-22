package subscriptions

import (
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// SetGroupVersionClientFactory replaces the group-version ASC client factory for tests.
// It returns a restore function to reset the previous factory.
func SetGroupVersionClientFactory(fn func() (*asc.Client, error)) func() {
	previous := subscriptionGroupVersionClientFactory
	if fn == nil {
		subscriptionGroupVersionClientFactory = shared.GetASCClient
	} else {
		subscriptionGroupVersionClientFactory = fn
	}
	return func() {
		subscriptionGroupVersionClientFactory = previous
	}
}
