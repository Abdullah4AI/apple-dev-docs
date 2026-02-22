package asc

import (
	"strings"
	"testing"
)

func TestAPIErrorError_SanitizesControlCharacters(t *testing.T) {
	err := &APIError{
		Title:  "Bad\x1b[31m",
		Detail: "Detail\x07",
		Code:   "CODE\x1b",
		AssociatedErrors: map[string][]APIAssociatedError{
			"/v1/resource\x1b[33m": {
				{
					Code:   "ENTITY_ERROR\x1b",
					Detail: "Associated detail\x07",
				},
			},
		},
	}

	message := err.Error()
	if strings.ContainsAny(message, "\x1b\x07") {
		t.Fatalf("expected control characters to be stripped, got %q", message)
	}
	if !strings.Contains(message, "Bad") || !strings.Contains(message, "Detail") {
		t.Fatalf("expected title and detail in message, got %q", message)
	}
	if !strings.Contains(message, "Associated detail") {
		t.Fatalf("expected associated detail in message, got %q", message)
	}
}

func TestAPIErrorError_AssociatedErrorsSortedByResourcePath(t *testing.T) {
	err := &APIError{
		Title:  "Cannot submit",
		Detail: "Fix associated errors",
		AssociatedErrors: map[string][]APIAssociatedError{
			"/v1/b": {
				{Detail: "B detail"},
			},
			"/v1/a": {
				{Detail: "A detail"},
			},
		},
	}

	message := err.Error()
	aIndex := strings.Index(message, "Associated errors for /v1/a:")
	bIndex := strings.Index(message, "Associated errors for /v1/b:")
	if aIndex == -1 || bIndex == -1 {
		t.Fatalf("expected associated error sections, got %q", message)
	}
	if aIndex > bIndex {
		t.Fatalf("expected associated errors to be sorted by path, got %q", message)
	}
}
