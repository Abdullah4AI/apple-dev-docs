package asc

import "fmt"

// DeveloperBundleIDCapabilityDisableResult is the stable receipt for a
// verified Developer Portal Bundle ID capability disable operation.
// Changed is false when the capability was already disabled and no PATCH was
// sent.
type DeveloperBundleIDCapabilityDisableResult struct {
	BundleID   string `json:"bundleId"`
	Capability string `json:"capability"`
	Enabled    bool   `json:"enabled"`
	Changed    bool   `json:"changed"`
	Status     string `json:"status"`
}

func developerBundleIDCapabilityDisableResultRows(result *DeveloperBundleIDCapabilityDisableResult) ([]string, [][]string) {
	headers := []string{"Bundle ID", "Capability", "Enabled", "Changed", "Status"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.BundleID,
		result.Capability,
		fmt.Sprintf("%t", result.Enabled),
		fmt.Sprintf("%t", result.Changed),
		result.Status,
	}}
}
