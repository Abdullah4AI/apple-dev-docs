package shared

import "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"

// ResolveAppStoreVersionState prefers the app version state when available.
func ResolveAppStoreVersionState(attrs asc.AppStoreVersionAttributes) string {
	if attrs.AppVersionState != "" {
		return attrs.AppVersionState
	}
	return attrs.AppStoreState
}
