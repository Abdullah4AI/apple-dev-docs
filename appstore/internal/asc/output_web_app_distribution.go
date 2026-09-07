package asc

import "fmt"

// WebAppDistributionSetResult is the receipt for an app-level distribution
// update. Status is unchanged when no PATCH was sent, verified after a
// successful read-back, and uncertain when Apple's write outcome cannot be
// established.
type WebAppDistributionSetResult struct {
	AppID                 string `json:"appId"`
	DistributionType      string `json:"distributionType"`
	EducationDiscountType string `json:"educationDiscountType"`
	Changed               bool   `json:"changed"`
	Verified              bool   `json:"verified"`
	Status                string `json:"status"`
}

func webAppDistributionSetRows(result *WebAppDistributionSetResult) ([]string, [][]string) {
	headers := []string{"App ID", "Distribution Type", "Education Discount Type", "Changed", "Verified", "Status"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.AppID,
		result.DistributionType,
		result.EducationDiscountType,
		fmt.Sprintf("%t", result.Changed),
		fmt.Sprintf("%t", result.Verified),
		result.Status,
	}}
}
