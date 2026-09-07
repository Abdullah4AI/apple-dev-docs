package asc

import (
	"reflect"
	"testing"
)

func TestWebReviewReplyResultUsesOutputRegistry(t *testing.T) {
	result := &WebReviewReplyResult{
		ThreadID:  "thread-1",
		DraftID:   "draft-1",
		MessageID: "message-1",
		Verified:  true,
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
		{"Reply", "Thread ID", "thread-1"},
		{"Reply", "Draft ID", "draft-1"},
		{"Reply", "Message ID", "message-1"},
		{"Reply", "Verified", "true"},
	}; !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}
