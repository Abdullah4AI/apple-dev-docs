package asc

import "fmt"

// WebWebsitePushIDMutationResult is the receipt for a verified modern
// Developer Portal Website Push ID mutation. It contains no session or
// credential material.
type WebWebsitePushIDMutationResult struct {
	Operation     string `json:"operation"`
	WebsitePushID string `json:"websitePushId"`
	Identifier    string `json:"identifier,omitempty"`
	Name          string `json:"name,omitempty"`
	Changed       bool   `json:"changed"`
	Verified      bool   `json:"verified"`
	Status        string `json:"status"`
}

func webWebsitePushIDMutationRows(result *WebWebsitePushIDMutationResult) ([]string, [][]string) {
	headers := []string{"Operation", "Website Push ID", "Identifier", "Name", "Changed", "Verified", "Status"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.Operation,
		result.WebsitePushID,
		result.Identifier,
		result.Name,
		fmt.Sprintf("%t", result.Changed),
		fmt.Sprintf("%t", result.Verified),
		result.Status,
	}}
}
