package asc

import "testing"

func TestRedactUploadOperationsWithholdsCapabilitiesWithoutMutatingInput(t *testing.T) {
	original := []UploadOperation{{
		Method: "PUT",
		URL:    "https://upload.example/object?Signature=signed-secret",
		Length: 42,
		Offset: 7,
		RequestHeaders: []HTTPHeader{{
			Name:  "Authorization",
			Value: "header-secret",
		}},
	}}

	safe := RedactUploadOperations(original)
	if len(safe) != 1 {
		t.Fatalf("redacted operations length = %d, want 1", len(safe))
	}
	if safe[0].Method != "PUT" || safe[0].Length != 42 || safe[0].Offset != 7 {
		t.Fatalf("redaction dropped non-sensitive operation metadata: %+v", safe[0])
	}
	if safe[0].URL != RedactedValuePlaceholder {
		t.Fatalf("redacted URL = %q, want placeholder", safe[0].URL)
	}
	if len(safe[0].RequestHeaders) != 1 || safe[0].RequestHeaders[0].Name != "Authorization" {
		t.Fatalf("redaction dropped header shape: %+v", safe[0].RequestHeaders)
	}
	if safe[0].RequestHeaders[0].Value != RedactedValuePlaceholder {
		t.Fatalf("redacted header value = %q, want placeholder", safe[0].RequestHeaders[0].Value)
	}
	if original[0].URL == safe[0].URL || original[0].RequestHeaders[0].Value == safe[0].RequestHeaders[0].Value {
		t.Fatal("redaction mutated the input operations")
	}
}
