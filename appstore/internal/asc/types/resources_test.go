package types

import (
	"encoding/json"
	"testing"
)

func TestResponseAccessors(t *testing.T) {
	r := &Response[struct{ Name string }]{
		Data: []Resource[struct{ Name string }]{
			{
				Type: ResourceTypeApps,
				ID:   "app-1",
				Attributes: struct{ Name string }{
					Name: "Example",
				},
			},
		},
		Links: Links{
			Self: "/v1/apps",
			Next: "/v1/apps?page=2",
		},
	}

	links := r.GetLinks()
	if links == nil || links.Next != "/v1/apps?page=2" {
		t.Fatalf("unexpected links: %+v", links)
	}

	data, ok := r.GetData().([]Resource[struct{ Name string }])
	if !ok {
		t.Fatalf("expected []Resource data type, got %T", r.GetData())
	}
	if len(data) != 1 || data[0].ID != "app-1" {
		t.Fatalf("unexpected data payload: %+v", data)
	}
}

func TestResponseGetMeta(t *testing.T) {
	r := &Response[struct{ Name string }]{
		Meta: json.RawMessage(`{"paging":{"total":42,"limit":5}}`),
	}

	meta := r.GetMeta()
	if string(meta) != `{"paging":{"total":42,"limit":5}}` {
		t.Fatalf("unexpected meta payload: %s", meta)
	}
	if total, ok := ParsePagingTotalOK(meta); !ok || total != 42 {
		t.Fatalf("expected paging total 42 from GetMeta, got %d (ok=%t)", total, ok)
	}

	empty := &Response[struct{ Name string }]{}
	if got := empty.GetMeta(); len(got) != 0 {
		t.Fatalf("expected empty meta, got %s", got)
	}
}

func TestResourcePreservesDecodedAttributesPresence(t *testing.T) {
	type attributes struct {
		Name string `json:"name,omitempty"`
	}

	tests := []struct {
		name       string
		input      string
		want       string
		attributes bool
	}{
		{
			name:       "absent",
			input:      `{"type":"apps","id":"app-1"}`,
			want:       `{"type":"apps","id":"app-1"}`,
			attributes: false,
		},
		{
			name:       "null",
			input:      `{"type":"apps","id":"app-1","attributes":null}`,
			want:       `{"type":"apps","id":"app-1","attributes":null}`,
			attributes: true,
		},
		{
			name:       "empty object",
			input:      `{"type":"apps","id":"app-1","attributes":{}}`,
			want:       `{"type":"apps","id":"app-1","attributes":{}}`,
			attributes: true,
		},
		{
			name:       "sparse object",
			input:      `{"type":"apps","id":"app-1","attributes":{"name":"Example"}}`,
			want:       `{"type":"apps","id":"app-1","attributes":{"name":"Example"}}`,
			attributes: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var resource Resource[attributes]
			if err := json.Unmarshal([]byte(tc.input), &resource); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got, err := json.Marshal(resource)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("output = %s, want %s", got, tc.want)
			}

			var fields map[string]json.RawMessage
			if err := json.Unmarshal(got, &fields); err != nil {
				t.Fatalf("decode output: %v", err)
			}
			_, present := fields["attributes"]
			if present != tc.attributes {
				t.Fatalf("attributes present = %t, want %t", present, tc.attributes)
			}
		})
	}
}

type nullAwareResourceAttributes struct {
	sawNull bool
}

func (a *nullAwareResourceAttributes) UnmarshalJSON(data []byte) error {
	a.sawNull = string(data) == "null"
	return nil
}

func TestResourceDecodesPresentNullAttributesIntoTypedValue(t *testing.T) {
	var resource Resource[nullAwareResourceAttributes]
	if err := json.Unmarshal([]byte(`{"type":"apps","id":"app-1","attributes":null}`), &resource); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resource.Attributes.sawNull {
		t.Fatal("expected the typed attributes decoder to receive null")
	}

	got, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `{"type":"apps","id":"app-1","attributes":null}` {
		t.Fatalf("output = %s", got)
	}
}

func TestResourceConstructedByCallerKeepsAttributes(t *testing.T) {
	resource := Resource[struct{}]{
		Type:       ResourceTypeApps,
		ID:         "app-1",
		Attributes: struct{}{},
	}
	got, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `{"type":"apps","id":"app-1","attributes":{}}` {
		t.Fatalf("output = %s", got)
	}
}

func TestLinkagesResponseAccessors(t *testing.T) {
	r := &LinkagesResponse{
		Data: []ResourceData{
			{Type: ResourceTypeBuilds, ID: "build-1"},
		},
		Links: Links{
			Self: "/v1/builds",
		},
	}

	links := r.GetLinks()
	if links == nil || links.Self != "/v1/builds" {
		t.Fatalf("unexpected links: %+v", links)
	}

	data, ok := r.GetData().([]ResourceData)
	if !ok {
		t.Fatalf("expected []ResourceData type, got %T", r.GetData())
	}
	if len(data) != 1 || data[0].ID != "build-1" {
		t.Fatalf("unexpected linkage payload: %+v", data)
	}
}

func TestTypeConstants(t *testing.T) {
	if PlatformIOS != "IOS" || PlatformMacOS != "MAC_OS" {
		t.Fatalf("unexpected platform constants: %q %q", PlatformIOS, PlatformMacOS)
	}
	if ChecksumAlgorithmSHA256 != "SHA_256" {
		t.Fatalf("unexpected checksum algorithm constant: %q", ChecksumAlgorithmSHA256)
	}
	if UTIIPA != "com.apple.ipa" || UTIPKG != "com.apple.pkg" {
		t.Fatalf("unexpected UTI constants: %q %q", UTIIPA, UTIPKG)
	}
}

func TestParsePagingTotalOK(t *testing.T) {
	tests := []struct {
		name      string
		meta      string
		wantTotal int
		wantOK    bool
	}{
		{
			name:      "nil meta",
			meta:      "",
			wantTotal: 0,
			wantOK:    false,
		},
		{
			name:      "meta missing total field",
			meta:      `{"paging":{"limit":1}}`,
			wantTotal: 0,
			wantOK:    false,
		},
		{
			name:      "meta with total zero",
			meta:      `{"paging":{"total":0,"limit":1}}`,
			wantTotal: 0,
			wantOK:    true,
		},
		{
			name:      "meta with positive total",
			meta:      `{"paging":{"total":42,"limit":1}}`,
			wantTotal: 42,
			wantOK:    true,
		},
		{
			name:      "invalid json",
			meta:      `not-json`,
			wantTotal: 0,
			wantOK:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var raw json.RawMessage
			if tc.meta != "" {
				raw = json.RawMessage(tc.meta)
			}
			got, ok := ParsePagingTotalOK(raw)
			if ok != tc.wantOK {
				t.Errorf("ParsePagingTotalOK() ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.wantTotal {
				t.Errorf("ParsePagingTotalOK() total = %d, want %d", got, tc.wantTotal)
			}
		})
	}
}

func TestParsePagingTotal_BackwardCompatibility(t *testing.T) {
	// Verify ParsePagingTotal still returns 0 in all absent-total cases.
	if got := ParsePagingTotal(nil); got != 0 {
		t.Errorf("ParsePagingTotal(nil) = %d, want 0", got)
	}
	if got := ParsePagingTotal(json.RawMessage(`{"paging":{"limit":1}}`)); got != 0 {
		t.Errorf("ParsePagingTotal(no total) = %d, want 0", got)
	}
	if got := ParsePagingTotal(json.RawMessage(`{"paging":{"total":7,"limit":1}}`)); got != 7 {
		t.Errorf("ParsePagingTotal(total=7) = %d, want 7", got)
	}
}

func TestRelationshipRequest_MarshalJSON_EncodesEmptyArray(t *testing.T) {
	// RelationshipRequest represents a to-many relationship payload. In JSON:API, an empty
	// relationship list is encoded as {"data":[]} (not {"data":null}).
	body, err := json.Marshal(RelationshipRequest{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got RelationshipRequest
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Data == nil {
		t.Fatalf("expected data to decode as an empty array, got nil (body=%q)", string(body))
	}
	if len(got.Data) != 0 {
		t.Fatalf("expected empty data array, got %d items", len(got.Data))
	}
}
