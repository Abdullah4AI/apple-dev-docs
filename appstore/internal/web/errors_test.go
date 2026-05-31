package web

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestAPIErrorRedactsRawBodyInErrorString(t *testing.T) {
	err := &APIError{
		Status:         422,
		AppleRequestID: "abc-request-id",
		CorrelationKey: "abc-correlation-key",
		rawBody:        []byte(`{"detail":"super-secret-token-123"}`),
	}
	message := err.Error()
	if strings.Contains(message, "super-secret-token-123") {
		t.Fatalf("expected redacted error string, got %q", message)
	}
	if !strings.Contains(message, "status 422") {
		t.Fatalf("expected status in error message, got %q", message)
	}
}

func TestIsDuplicateAppNameError(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		wantDup bool
	}{
		{
			name: "duplicate by code and detail",
			err: &APIError{rawBody: []byte(`{
				"errors":[{
					"code":"ENTITY_ERROR.ATTRIBUTE.INVALID.DUPLICATE.DIFFERENT_ACCOUNT",
					"detail":"The app name you entered is already being used."
				}]
			}`)},
			wantDup: true,
		},
		{
			name: "non-duplicate code",
			err: &APIError{rawBody: []byte(`{
				"errors":[{"code":"ENTITY_ERROR.ATTRIBUTE.INVALID","detail":"invalid value"}]
			}`)},
			wantDup: false,
		},
		{
			name:    "non api error",
			err:     errors.New("nope"),
			wantDup: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsDuplicateAppNameError(tc.err); got != tc.wantDup {
				t.Fatalf("IsDuplicateAppNameError()=%v want %v", got, tc.wantDup)
			}
		})
	}
}

func TestIsAlreadyExistsConflict(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "already exists code",
			err: &APIError{
				Status: http.StatusConflict,
				rawBody: []byte(`{
					"errors":[{
						"code":"ENTITY_ERROR.ATTRIBUTE.INVALID.ALREADY_EXISTS",
						"title":"The request entity conflicts with the current state."
					}]
				}`),
			},
			want: true,
		},
		{
			name: "already attached detail",
			err: &APIError{
				Status:  http.StatusConflict,
				rawBody: []byte(`{"errors":[{"detail":"This in-app purchase is already attached to the app version."}]}`),
			},
			want: false,
		},
		{
			name: "already exists for another target",
			err: &APIError{
				Status: http.StatusConflict,
				rawBody: []byte(`{
						"errors":[{
							"code":"ENTITY_ERROR.ATTRIBUTE.INVALID.ALREADY_EXISTS",
							"detail":"This in-app purchase is already attached to another submission."
						}]
					}`),
			},
			want: false,
		},
		{
			name: "mixed already exists and blocking conflict",
			err: &APIError{
				Status: http.StatusConflict,
				rawBody: []byte(`{
					"errors":[{
						"code":"ENTITY_ERROR.ATTRIBUTE.INVALID.ALREADY_EXISTS",
						"title":"The request entity conflicts with the current state."
					},{
						"code":"STATE_ERROR.INVALID",
						"detail":"This in-app purchase cannot be attached in the current state."
					}]
				}`),
			},
			want: false,
		},
		{
			name: "other conflict",
			err: &APIError{
				Status:  http.StatusConflict,
				rawBody: []byte(`{"errors":[{"code":"STATE_ERROR","detail":"Invalid state transition."}]}`),
			},
			want: false,
		},
		{
			name: "non conflict",
			err: &APIError{
				Status:  http.StatusBadRequest,
				rawBody: []byte(`{"errors":[{"code":"ENTITY_ERROR.ATTRIBUTE.INVALID.ALREADY_EXISTS"}]}`),
			},
			want: false,
		},
		{
			name: "wrapped",
			err: fmt.Errorf("wrapped: %w", &APIError{
				Status:  http.StatusConflict,
				rawBody: []byte(`already exists`),
			}),
			want: true,
		},
		{
			name: "non api error",
			err:  errors.New("nope"),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsAlreadyExistsConflict(tc.err); got != tc.want {
				t.Fatalf("IsAlreadyExistsConflict()=%v want %v", got, tc.want)
			}
		})
	}
}
