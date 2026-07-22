package iap

import "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"

// SetVersionClientFactory replaces the IAP version ASC client factory for tests.
func SetVersionClientFactory(fn func() (*asc.Client, error)) func() {
	previousVersion := iapVersionClientFactory
	previousQuery := iapQueryClientFactory
	if fn == nil {
		iapVersionClientFactory = previousVersion
		iapQueryClientFactory = previousQuery
	} else {
		iapVersionClientFactory = fn
		iapQueryClientFactory = fn
	}
	return func() {
		iapVersionClientFactory = previousVersion
		iapQueryClientFactory = previousQuery
	}
}
