package asc

import (
	"encoding/json"
	"testing"
)

func TestBetaGroupsResponsePreservesSparseAttributeFields(t *testing.T) {
	input := []byte(`{"data":[{"type":"betaGroups","id":"group-1","attributes":{"publicLinkEnabled":false}}],"links":{}}`)

	var response BetaGroupsResponse
	if err := json.Unmarshal(input, &response); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var envelope struct {
		Data []struct {
			Attributes map[string]json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(envelope.Data) != 1 {
		t.Fatalf("data count = %d, want 1", len(envelope.Data))
	}
	attributes := envelope.Data[0].Attributes
	if len(attributes) != 1 {
		t.Fatalf("attributes = %s, want only publicLinkEnabled", encoded)
	}
	if got := string(attributes["publicLinkEnabled"]); got != "false" {
		t.Fatalf("publicLinkEnabled = %s, want false", got)
	}
	if _, present := attributes["name"]; present {
		t.Fatalf("output invented name: %s", encoded)
	}
}
