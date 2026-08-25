package asc

import (
	"fmt"
	"strings"
)

// AvailabilityPlatformListing summarizes the selected App Store version for
// one platform.
type AvailabilityPlatformListing struct {
	Platform      string `json:"platform"`
	VersionString string `json:"versionString"`
	State         string `json:"state"`
	Live          bool   `json:"live"`
	StateKnown    bool   `json:"stateKnown"`
	CreatedDate   string `json:"createdDate,omitempty"`
}

// AvailabilityPlatformsResult summarizes per-platform App Store listings for
// an app.
type AvailabilityPlatformsResult struct {
	AppID     string                        `json:"appId"`
	Platforms []AvailabilityPlatformListing `json:"platforms"`
}

// AvailabilityRemoveFromSaleResult summarizes a verified remove-from-sale
// operation.
type AvailabilityRemoveFromSaleResult struct {
	AppID                             string                        `json:"appId"`
	AvailabilityID                    string                        `json:"availabilityId"`
	Status                            string                        `json:"status"`
	AvailableInNewTerritories         bool                          `json:"availableInNewTerritories"`
	TotalTerritories                  int                           `json:"totalTerritories"`
	UpdatedTerritories                int                           `json:"updatedTerritories"`
	AlreadyUnavailableTerritories     int                           `json:"alreadyUnavailableTerritories"`
	VerifiedUnavailableTerritories    int                           `json:"verifiedUnavailableTerritories"`
	FailedTerritories                 []string                      `json:"failedTerritories"`
	PlatformListingsVerified          bool                          `json:"platformListingsVerified"`
	PlatformListingsVerificationError string                        `json:"platformListingsVerificationError,omitempty"`
	RemovedPlatformListings           []AvailabilityPlatformListing `json:"removedPlatformListings,omitempty"`
}

func availabilityPlatformsResultRows(result *AvailabilityPlatformsResult) ([]string, [][]string) {
	headers := []string{"Platform", "Version", "State", "Live", "State known", "Created"}
	rows := make([][]string, 0, len(result.Platforms))
	for _, listing := range result.Platforms {
		rows = append(rows, []string{
			listing.Platform,
			listing.VersionString,
			listing.State,
			fmt.Sprintf("%t", listing.Live),
			fmt.Sprintf("%t", listing.StateKnown),
			listing.CreatedDate,
		})
	}
	return headers, rows
}

func availabilityRemoveFromSaleResultRows(result *AvailabilityRemoveFromSaleResult) ([]string, [][]string) {
	return []string{"Field", "Value"}, [][]string{
		{"App ID", result.AppID},
		{"Availability ID", result.AvailabilityID},
		{"Status", result.Status},
		{"Available in new territories", fmt.Sprintf("%t", result.AvailableInNewTerritories)},
		{"Total territories", fmt.Sprintf("%d", result.TotalTerritories)},
		{"Updated territories", fmt.Sprintf("%d", result.UpdatedTerritories)},
		{"Already unavailable", fmt.Sprintf("%d", result.AlreadyUnavailableTerritories)},
		{"Verified unavailable", fmt.Sprintf("%d", result.VerifiedUnavailableTerritories)},
		{"Failed territories", strings.Join(result.FailedTerritories, ", ")},
		{"Platform listings verified", fmt.Sprintf("%t", result.PlatformListingsVerified)},
		{"Platform listings verification error", result.PlatformListingsVerificationError},
		{"Removed platform listings", availabilityPlatformListingSummary(result.RemovedPlatformListings)},
	}
}

func availabilityPlatformListingSummary(listings []AvailabilityPlatformListing) string {
	values := make([]string, 0, len(listings))
	for _, listing := range listings {
		value := strings.TrimSpace(listing.Platform)
		if version := strings.TrimSpace(listing.VersionString); version != "" {
			if value != "" {
				value += " "
			}
			value += version
		}
		if value != "" {
			values = append(values, value)
		}
	}
	return strings.Join(values, ", ")
}
