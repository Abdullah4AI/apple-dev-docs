package cmdtest

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewItemsAddIncludesAssociatedErrorsInReturnedError(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/v1/reviewSubmissionItems" {
			t.Fatalf("expected path /v1/reviewSubmissionItems, got %s", req.URL.Path)
		}

		body := `{
			"errors": [{
				"status": "409",
				"code": "STATE_ERROR.ENTITY_STATE_INVALID",
				"title": "appStoreVersions with id 'version-1' is not in valid state.",
				"detail": "This resource cannot be reviewed, please check associated errors to see why.",
				"meta": {
					"associatedErrors": {
						"/v1/ageRatingDeclarations/age-rating-1": [
							{
								"code": "ENTITY_ERROR.ATTRIBUTE.REQUIRED",
								"detail": "You must provide a value for the attribute 'parentalControls' with this request"
							},
							{
								"code": "ENTITY_ERROR.ATTRIBUTE.REQUIRED",
								"detail": "You must provide a value for the attribute 'healthOrWellnessTopics' with this request"
							}
						]
					}
				}
			}]
		}`

		return &http.Response{
			StatusCode: http.StatusConflict,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"review", "items-add", "--submission", "submission-1", "--item-type", "appStoreVersions", "--item-id", "version-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if runErr == nil {
		t.Fatal("expected error, got nil")
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}

	message := runErr.Error()
	if !strings.Contains(message, "review items-add: appStoreVersions with id 'version-1' is not in valid state.") {
		t.Fatalf("expected wrapped top-level message, got %q", message)
	}
	if !strings.Contains(message, "Associated errors for /v1/ageRatingDeclarations/age-rating-1:") {
		t.Fatalf("expected associated error heading, got %q", message)
	}
	if !strings.Contains(message, "parentalControls") {
		t.Fatalf("expected parentalControls detail, got %q", message)
	}
	if !strings.Contains(message, "healthOrWellnessTopics") {
		t.Fatalf("expected healthOrWellnessTopics detail, got %q", message)
	}
}
