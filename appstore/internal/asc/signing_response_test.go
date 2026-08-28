package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBundleIDsResponseMarshalPreservesAttributesPresence(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMember bool
	}{
		{name: "omitted", input: `{"type":"bundleIds","id":"bundle-1"}`, wantMember: false},
		{name: "empty", input: `{"type":"bundleIds","id":"bundle-1","attributes":{}}`, wantMember: true},
		{name: "populated", input: `{"type":"bundleIds","id":"bundle-1","attributes":{"name":"Example"}}`, wantMember: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var response BundleIDsResponse
			if err := json.Unmarshal([]byte(`{"data":[`+test.input+`],"links":{}}`), &response); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			encoded, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			hasMember := strings.Contains(string(encoded), `"attributes"`)
			if hasMember != test.wantMember {
				t.Fatalf("encoded = %s, attributes present = %t, want %t", encoded, hasMember, test.wantMember)
			}
		})
	}
}

func TestBundleIDsResponseMarshalPreservesDataNullability(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "null", input: `{"data":null,"links":{}}`, want: `"data":null`},
		{name: "empty", input: `{"data":[],"links":{}}`, want: `"data":[]`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var response BundleIDsResponse
			if err := json.Unmarshal([]byte(test.input), &response); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			encoded, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !strings.Contains(string(encoded), test.want) {
				t.Fatalf("encoded = %s, want member %s", encoded, test.want)
			}
		})
	}
}

func TestBundleIDsResponseMarshalPreservesSparseAttributeFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "null", input: `null`, want: `null`},
		{name: "empty object", input: `{}`, want: `{}`},
		{name: "name only", input: `{"name":"Example"}`, want: `{"name":"Example"}`},
		{name: "explicit empty identifier", input: `{"identifier":""}`, want: `{"identifier":""}`},
		{name: "null seed ID", input: `{"seedId":null}`, want: `{"seedId":null}`},
		{name: "platform and empty seed ID", input: `{"platform":"IOS","seedId":""}`, want: `{"platform":"IOS","seedId":""}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var response BundleIDsResponse
			input := `{"data":[{"type":"bundleIds","id":"bundle-1","attributes":` + test.input + `}],"links":{}}`
			if err := json.Unmarshal([]byte(input), &response); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			encoded, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var payload struct {
				Data []struct {
					Attributes json.RawMessage `json:"attributes"`
				} `json:"data"`
			}
			if err := json.Unmarshal(encoded, &payload); err != nil {
				t.Fatalf("decode encoded response: %v", err)
			}
			if got := string(payload.Data[0].Attributes); got != test.want {
				t.Fatalf("attributes = %s, want %s", got, test.want)
			}
		})
	}
}

func TestBundleIDsResponseMarshalUsesMutatedSparseAttributeValues(t *testing.T) {
	var response BundleIDsResponse
	input := `{"data":[{"type":"bundleIds","id":"bundle-1","attributes":{"name":"Before"}}],"links":{}}`
	if err := json.Unmarshal([]byte(input), &response); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	response.Data[0].Attributes.Name = "After"

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"attributes":{"name":"After"}`) {
		t.Fatalf("encoded = %s, want mutated sparse attributes", encoded)
	}
}
