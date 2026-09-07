package asc

import (
	"reflect"
	"testing"
)

func TestWebReviewDraftResultUsesOutputRegistry(t *testing.T) {
	result := &WebReviewDraftResult{
		AppID:    "app-1",
		ThreadID: "thread-1",
		DraftID:  "draft-1",
		Action:   "updated",
		Verified: true,
	}

	var headers []string
	var rows [][]string
	if err := renderByRegistry(result, func(gotHeaders []string, gotRows [][]string) {
		headers = gotHeaders
		rows = gotRows
	}); err != nil {
		t.Fatalf("renderByRegistry() error: %v", err)
	}
	if want := []string{"Section", "Field", "Value"}; !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers = %#v, want %#v", headers, want)
	}
	if want := [][]string{
		{"Draft", "App ID", "app-1"},
		{"Draft", "Thread ID", "thread-1"},
		{"Draft", "Draft ID", "draft-1"},
		{"Draft", "Action", "updated"},
		{"Draft", "Verified", "true"},
	}; !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}
