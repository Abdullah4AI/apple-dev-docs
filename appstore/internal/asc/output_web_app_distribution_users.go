package asc

import "fmt"

// WebAppDistributionUserMutationResult is the receipt for one app-scoped
// custom-distribution Apple Account recipient mutation. A nil Changed value
// means that an attempted write had an uncertain outcome.
type WebAppDistributionUserMutationResult struct {
	AppID       string `json:"appId"`
	RecipientID string `json:"recipientId,omitempty"`
	AppleID     string `json:"appleId"`
	Changed     *bool  `json:"changed"`
	Verified    bool   `json:"verified"`
	Status      string `json:"status"`
}

func webAppDistributionUserMutationRows(result *WebAppDistributionUserMutationResult) ([]string, [][]string) {
	headers := []string{"App ID", "Recipient ID", "Apple Account", "Changed", "Verified", "Status"}
	if result == nil {
		return headers, nil
	}
	changed := "unknown"
	if result.Changed != nil {
		changed = fmt.Sprintf("%t", *result.Changed)
	}
	recipientID := result.RecipientID
	if recipientID == "" {
		recipientID = "unknown"
	}
	appleID := result.AppleID
	if appleID == "" {
		appleID = "unknown"
	}
	return headers, [][]string{{
		result.AppID,
		recipientID,
		appleID,
		changed,
		fmt.Sprintf("%t", result.Verified),
		result.Status,
	}}
}
