package cmd

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/storekit"
	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

func TestExitCodeFromError_WebAPIStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected int
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, expected: ExitAuth},
		{name: "forbidden", status: http.StatusForbidden, expected: ExitAuth},
		{name: "not found", status: http.StatusNotFound, expected: ExitNotFound},
		{name: "conflict", status: http.StatusConflict, expected: ExitConflict},
		{name: "server error", status: http.StatusInternalServerError, expected: ExitHTTPInternalServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("web command failed: %w", &webcore.APIError{Status: tt.status})
			if got := ExitCodeFromError(err); got != tt.expected {
				t.Fatalf("ExitCodeFromError() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestExitCodeFromError_StoreKitAPIStatusRemainsGeneric(t *testing.T) {
	err := fmt.Errorf("storekit command failed: %w", &storekit.APIError{StatusCode: http.StatusInternalServerError})
	if got := ExitCodeFromError(err); got != ExitError {
		t.Fatalf("ExitCodeFromError() = %d, want %d", got, ExitError)
	}
}
