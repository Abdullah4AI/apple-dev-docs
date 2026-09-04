package asc

import (
	"fmt"
	"strings"
)

// WebAppGroupDeleteResult is the receipt for a verified Developer Portal App
// Group deletion.
type WebAppGroupDeleteResult struct {
	GroupID    string `json:"groupId"`
	Identifier string `json:"identifier"`
	Name       string `json:"name,omitempty"`
	Deleted    bool   `json:"deleted"`
	Status     string `json:"status"`
}

// WebAppGroupUnassignResult is the receipt for removing one App Group from a
// Bundle ID. Changed is false when the group was not assigned and no write was
// sent.
type WebAppGroupUnassignResult struct {
	BundleID          string   `json:"bundleId"`
	GroupID           string   `json:"groupId"`
	RemainingGroupIDs []string `json:"remainingGroupIds"`
	Changed           bool     `json:"changed"`
	Status            string   `json:"status"`
}

// WebAppGroupSetResult is the diff receipt for converging a Bundle ID on a
// desired App Group set. Changed is false when the current set already matched
// and no write was sent.
type WebAppGroupSetResult struct {
	BundleID string   `json:"bundleId"`
	GroupIDs []string `json:"groupIds"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Changed  bool     `json:"changed"`
	Status   string   `json:"status"`
}

func webAppGroupDeleteRows(result *WebAppGroupDeleteResult) ([]string, [][]string) {
	headers := []string{"Group ID", "Name", "Identifier", "Deleted", "Status"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{result.GroupID, webAppGroupText(result.Name), webAppGroupText(result.Identifier), fmt.Sprintf("%t", result.Deleted), result.Status}}
}

func webAppGroupUnassignRows(result *WebAppGroupUnassignResult) ([]string, [][]string) {
	headers := []string{"Bundle ID", "Group ID", "Remaining Group IDs", "Changed", "Status"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{result.BundleID, result.GroupID, joinWebAppGroupIDs(result.RemainingGroupIDs), fmt.Sprintf("%t", result.Changed), result.Status}}
}

func webAppGroupSetRows(result *WebAppGroupSetResult) ([]string, [][]string) {
	headers := []string{"Bundle ID", "Group IDs", "Added", "Removed", "Changed", "Status"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{result.BundleID, joinWebAppGroupIDs(result.GroupIDs), joinWebAppGroupIDs(result.Added), joinWebAppGroupIDs(result.Removed), fmt.Sprintf("%t", result.Changed), result.Status}}
}

func webAppGroupText(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func joinWebAppGroupIDs(ids []string) string {
	if len(ids) == 0 {
		return "-"
	}
	return strings.Join(ids, ", ")
}
