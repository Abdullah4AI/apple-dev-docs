package shared

import (
	"fmt"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

// SelectBestAppInfoID chooses the most editable app info for updates.
func SelectBestAppInfoID(appInfos *asc.AppInfosResponse) string {
	if appInfos == nil || len(appInfos.Data) == 0 {
		return ""
	}

	const target = "PREPARE_FOR_SUBMISSION"

	var firstNonLive string
	for _, info := range appInfos.Data {
		state := strings.ToUpper(appInfoAttrString(info.Attributes, "state"))
		appStoreState := strings.ToUpper(appInfoAttrString(info.Attributes, "appStoreState"))

		if state == target || appStoreState == target {
			return info.ID
		}
		if firstNonLive == "" && isNonLiveAppInfoState(state, appStoreState) {
			firstNonLive = info.ID
		}
	}
	if firstNonLive != "" {
		return firstNonLive
	}
	return appInfos.Data[0].ID
}

// isNonLiveAppInfoState reports whether either state indicates a non-live app info.
func isNonLiveAppInfoState(state, appStoreState string) bool {
	isLive := func(value string) bool {
		switch value {
		case "READY_FOR_DISTRIBUTION", "READY_FOR_SALE":
			return true
		default:
			return false
		}
	}

	if state != "" && !isLive(state) {
		return true
	}
	if appStoreState != "" && !isLive(appStoreState) {
		return true
	}
	return false
}

// appInfoAttrString fetches a string attribute from the App Info payload.
func appInfoAttrString(attrs asc.AppInfoAttributes, key string) string {
	if attrs == nil {
		return ""
	}
	value, ok := attrs[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
