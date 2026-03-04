package shared

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestMapTerritoryAvailabilityIDs(t *testing.T) {
	relationships := asc.TerritoryAvailabilityRelationships{
		Territory: asc.Relationship{
			Data: asc.ResourceData{
				Type: asc.ResourceTypeTerritories,
				ID:   "usa",
			},
		},
	}
	relationshipsJSON, err := json.Marshal(relationships)
	if err != nil {
		t.Fatalf("failed to marshal relationships: %v", err)
	}

	resp := &asc.TerritoryAvailabilitiesResponse{
		Data: []asc.Resource[asc.TerritoryAvailabilityAttributes]{
			{
				Type:          asc.ResourceTypeTerritoryAvailabilities,
				ID:            "ta-1",
				Relationships: relationshipsJSON,
			},
		},
	}

	ids, err := MapTerritoryAvailabilityIDs(resp)
	if err != nil {
		t.Fatalf("MapTerritoryAvailabilityIDs() error: %v", err)
	}
	if ids["USA"] != "ta-1" {
		t.Fatalf("expected territory USA to map to ta-1, got %q", ids["USA"])
	}
}

func TestMapTerritoryAvailabilityIDs_FallbackID(t *testing.T) {
	payload := `{"s":"6740467361","t":"USA"}`
	encoded := base64.RawStdEncoding.EncodeToString([]byte(payload))

	resp := &asc.TerritoryAvailabilitiesResponse{
		Data: []asc.Resource[asc.TerritoryAvailabilityAttributes]{
			{
				Type: asc.ResourceTypeTerritoryAvailabilities,
				ID:   encoded,
			},
		},
	}

	ids, err := MapTerritoryAvailabilityIDs(resp)
	if err != nil {
		t.Fatalf("MapTerritoryAvailabilityIDs() error: %v", err)
	}
	if ids["USA"] != encoded {
		t.Fatalf("expected territory USA to map to %q, got %q", encoded, ids["USA"])
	}
}

func TestMapTerritoryAvailabilityIDs_NilResponse(t *testing.T) {
	_, err := MapTerritoryAvailabilityIDs(nil)
	if err == nil {
		t.Fatal("expected error for nil response")
	}
}
