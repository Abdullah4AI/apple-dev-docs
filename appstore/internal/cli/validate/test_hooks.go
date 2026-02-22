package validate

import (
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// SetClientFactory replaces the ASC client factory for tests.
// It returns a restore function to reset the previous handler.
func SetClientFactory(fn func() (*asc.Client, error)) func() {
	previous := clientFactory
	if fn == nil {
		clientFactory = shared.GetASCClient
	} else {
		clientFactory = fn
	}
	return func() {
		clientFactory = previous
	}
}
