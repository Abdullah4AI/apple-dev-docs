package pricing

import (
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// SetAvailabilityClientFactory replaces the ASC client factory for availability tests.
// It returns a restore function to reset the previous handler.
func SetAvailabilityClientFactory(fn func() (*asc.Client, error)) func() {
	previous := pricingAvailabilityClientFactory
	if fn == nil {
		pricingAvailabilityClientFactory = shared.GetASCClient
	} else {
		pricingAvailabilityClientFactory = fn
	}
	return func() {
		pricingAvailabilityClientFactory = previous
	}
}
