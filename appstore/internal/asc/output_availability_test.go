package asc

import (
	"reflect"
	"testing"
)

func TestAvailabilityPlatformsResultUsesOutputRegistry(t *testing.T) {
	result := &AvailabilityPlatformsResult{
		AppID: "app-1",
		Platforms: []AvailabilityPlatformListing{{
			Platform:      "IOS",
			VersionString: "4.2.0",
			State:         "READY_FOR_DISTRIBUTION",
			Live:          true,
			StateKnown:    true,
			CreatedDate:   "2026-08-19T00:00:00Z",
		}},
	}

	var headers []string
	var rows [][]string
	if err := renderByRegistry(result, func(gotHeaders []string, gotRows [][]string) {
		headers = gotHeaders
		rows = gotRows
	}); err != nil {
		t.Fatalf("renderByRegistry() error: %v", err)
	}
	if want := []string{"Platform", "Version", "State", "Live", "State known", "Created"}; !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers = %#v, want %#v", headers, want)
	}
	if want := [][]string{{"IOS", "4.2.0", "READY_FOR_DISTRIBUTION", "true", "true", "2026-08-19T00:00:00Z"}}; !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func TestAvailabilityRemoveFromSaleResultUsesOutputRegistry(t *testing.T) {
	result := &AvailabilityRemoveFromSaleResult{
		AppID:                          "app-1",
		AvailabilityID:                 "availability-1",
		Status:                         "removedFromSale",
		AvailableInNewTerritories:      false,
		TotalTerritories:               2,
		UpdatedTerritories:             1,
		AlreadyUnavailableTerritories:  1,
		VerifiedUnavailableTerritories: 2,
		FailedTerritories:              []string{},
		RemovedPlatformListings: []AvailabilityPlatformListing{{
			Platform:      "IOS",
			VersionString: "4.2.0",
			Live:          true,
			StateKnown:    true,
		}},
		PlatformListingsVerified: true,
	}

	var headers []string
	var rows [][]string
	if err := renderByRegistry(result, func(gotHeaders []string, gotRows [][]string) {
		headers = gotHeaders
		rows = gotRows
	}); err != nil {
		t.Fatalf("renderByRegistry() error: %v", err)
	}
	if want := []string{"Field", "Value"}; !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers = %#v, want %#v", headers, want)
	}
	if len(rows) != 12 || rows[1][1] != "availability-1" || rows[7][1] != "2" || rows[9][1] != "true" || rows[11][1] != "IOS 4.2.0" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}
