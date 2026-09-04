package signing

import (
	"strings"
	"testing"
)

func TestRejectDuplicateSigningRunJSONKeysFoldsCaseVariants(t *testing.T) {
	err := rejectDuplicateSigningRunJSONKeys([]byte(`{"bundleId":"a","BundleID":"b"}`))
	if err == nil || !strings.Contains(err.Error(), "duplicate JSON field") {
		t.Fatalf("rejectDuplicateSigningRunJSONKeys() error = %v, want case-variant duplicate rejection", err)
	}
	if err := rejectDuplicateSigningRunJSONKeys([]byte(`{"bundleId":"a","profilePath":"b"}`)); err != nil {
		t.Fatalf("rejectDuplicateSigningRunJSONKeys() error = %v, want distinct keys accepted", err)
	}
	if err := rejectDuplicateSigningRunJSONKeys([]byte(`{"a":{"x":1},"b":{"X":2,"x":3}}`)); err == nil {
		t.Fatal("rejectDuplicateSigningRunJSONKeys() missed a nested case-variant duplicate")
	}
	if err := rejectDuplicateSigningRunJSONKeys([]byte(`{"a":{"x":1},"b":{"x":2}}`)); err != nil {
		t.Fatalf("rejectDuplicateSigningRunJSONKeys() error = %v, want sibling objects folded independently", err)
	}
}

func TestRejectDuplicateSigningRunJSONKeysMatchesEqualFoldSemantics(t *testing.T) {
	// The Kelvin sign folds to ASCII k under the simple folding that
	// encoding/json uses for field matching.
	if err := rejectDuplicateSigningRunJSONKeys([]byte("{\"Key\":1,\"key\":2}")); err == nil {
		t.Fatal("rejectDuplicateSigningRunJSONKeys() missed a Unicode simple-fold duplicate")
	}
}
