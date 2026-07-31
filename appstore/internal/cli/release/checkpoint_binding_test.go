package release

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/metadata"
	validatecli "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/validate"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/validation"
)

func newCheckpointBindingClient(t *testing.T, handler releaseRoundTripFunc) *asc.Client {
	t.Helper()

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})
	http.DefaultTransport = handler
	return newReleaseTestClient(t)
}

func checkpointBindingOptions() runOptions {
	return runOptions{
		AppID:           "APP_123",
		Version:         "2.4.0",
		BuildID:         "BUILD_123",
		Platform:        "IOS",
		Mode:            releaseModeRun,
		SubmitForReview: true,
	}
}

func TestVerifyResumedCheckpointBindingRejectsVersionOwnedByAnotherApp(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_B":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_B","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_B"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_B",
		Completed: map[string]bool{stepEnsureVersion: true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject a version owned by another app")
	}
	if !strings.Contains(err.Error(), "belongs to app") {
		t.Fatalf("expected ownership error, got %v", err)
	}
}

func TestVerifyResumedCheckpointBindingRejectsVersionStringMismatch(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"9.9.9","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{stepEnsureVersion: true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject a version string mismatch")
	}
	if !strings.Contains(err.Error(), "2.4.0") {
		t.Fatalf("expected error naming the requested version, got %v", err)
	}
}

func TestVerifyResumedCheckpointBindingRejectsCompletedStepsWithoutVersionBinding(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
	})

	checkpoint := runCheckpoint{
		Completed: map[string]bool{stepApplyMetadata: true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject completed steps without a bound version")
	}
}

// TestVerifyResumedCheckpointBindingExplainsMissingSubmitForReviewFlag proves
// that resuming a checkpoint with a completed submit_review step but without
// --submit-for-review reports the flag mismatch instead of claiming the
// checkpoint records an unknown step.
func TestVerifyResumedCheckpointBindingExplainsMissingSubmitForReviewFlag(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
	})

	opts := checkpointBindingOptions()
	opts.SubmitForReview = false
	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_123",
		Completed:    map[string]bool{stepSubmitReview: true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, opts, &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject the missing --submit-for-review flag")
	}
	if !strings.Contains(err.Error(), "--submit-for-review") {
		t.Fatalf("expected error naming --submit-for-review, got %v", err)
	}
	if strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected a flag mismatch error, not an unknown-step error, got %v", err)
	}
}

func TestVerifyResumedCheckpointBindingRejectsUnknownCompletedStep(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{"publish_everything": true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject an unknown completed step")
	}
	if !strings.Contains(err.Error(), "publish_everything") {
		t.Fatalf("expected error naming the unknown step, got %v", err)
	}
}

func TestVerifyResumedCheckpointBindingDropsUnprovenAttachBuildCompletion(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_OTHER","attributes":{"version":"41"}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{stepEnsureVersion: true, stepAttachBuild: true},
	}
	var messages []string
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, func(message string) {
		messages = append(messages, message)
	}); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if checkpoint.Completed[stepAttachBuild] {
		t.Fatal("expected unproven attach_build completion to be discarded")
	}
	if !checkpoint.Completed[stepEnsureVersion] {
		t.Fatal("expected verified ensure_version completion to survive")
	}
	if len(messages) == 0 || !strings.Contains(strings.Join(messages, "\n"), stepAttachBuild) {
		t.Fatalf("expected a diagnostic naming attach_build, got %v", messages)
	}
}

// TestVerifyResumedCheckpointBindingDropsReadinessWithUnprovenAttachBuild
// proves that discarding an attach_build completion also discards a completed
// validate_readiness: readiness was checked against whatever build was
// attached at the time, so it must run again after the build is re-attached.
func TestVerifyResumedCheckpointBindingDropsReadinessWithUnprovenAttachBuild(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_OTHER","attributes":{"version":"41"}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{
			stepEnsureVersion:     true,
			stepAttachBuild:       true,
			stepValidateReadiness: true,
		},
	}
	var messages []string
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, func(message string) {
		messages = append(messages, message)
	}); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if checkpoint.Completed[stepAttachBuild] {
		t.Fatal("expected unproven attach_build completion to be discarded")
	}
	if checkpoint.Completed[stepValidateReadiness] {
		t.Fatal("expected dependent validate_readiness completion to be discarded")
	}
	if !checkpoint.Completed[stepEnsureVersion] {
		t.Fatal("expected verified ensure_version completion to survive")
	}
	if len(messages) == 0 || !strings.Contains(strings.Join(messages, "\n"), stepValidateReadiness) {
		t.Fatalf("expected a diagnostic naming validate_readiness, got %v", messages)
	}
}

func TestVerifyResumedCheckpointBindingDropsSubmissionNotBoundToVersion(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_OTHER":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"reviewSubmissions","id":"SUBMISSION_OTHER","attributes":{"state":"WAITING_FOR_REVIEW","platform":"IOS"},"relationships":{"appStoreVersionForReview":{"data":{"type":"appStoreVersions","id":"VERSION_OTHER"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_OTHER",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepAttachBuild:   true,
			stepSubmitReview:  true,
		},
	}
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if checkpoint.Completed[stepSubmitReview] {
		t.Fatal("expected unproven submit_review completion to be discarded")
	}
	if checkpoint.SubmissionID != "" {
		t.Fatalf("expected unproven submission ID to be cleared, got %q", checkpoint.SubmissionID)
	}
}

// TestVerifyResumedCheckpointBindingResolvesSubmissionVersionFromItems proves
// that a proven-submitted checkpoint survives when the submission response
// omits the appStoreVersionForReview linkage (a plain GET may not include it):
// binding is then re-derived from the submission's items.
func TestVerifyResumedCheckpointBindingResolvesSubmissionVersionFromItems(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"reviewSubmissions","id":"SUBMISSION_123","attributes":{"state":"WAITING_FOR_REVIEW","platform":"IOS"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_123/items":
			return releaseJSONResponse(http.StatusOK, `{"data":[{"type":"reviewSubmissionItems","id":"ITEM_1","relationships":{"appStoreVersion":{"data":{"type":"appStoreVersions","id":"VERSION_123"}}}}],"links":{}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_123",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepAttachBuild:   true,
			stepSubmitReview:  true,
		},
	}
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if !checkpoint.Completed[stepSubmitReview] {
		t.Fatal("expected item-proven submit_review completion to survive a missing relationship linkage")
	}
	if checkpoint.SubmissionID != "SUBMISSION_123" {
		t.Fatalf("expected submission ID to survive, got %q", checkpoint.SubmissionID)
	}
}

// TestVerifyResumedCheckpointBindingScansAllItemsForResumedVersion proves the
// item fallback searches for the checkpoint's version instead of trusting the
// first item that carries any version: a submission holding another item ahead
// of the resumed version must still count as bound.
func TestVerifyResumedCheckpointBindingScansAllItemsForResumedVersion(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"reviewSubmissions","id":"SUBMISSION_123","attributes":{"state":"WAITING_FOR_REVIEW","platform":"IOS"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_123/items":
			return releaseJSONResponse(http.StatusOK, `{"data":[{"type":"reviewSubmissionItems","id":"ITEM_OTHER","relationships":{"appStoreVersion":{"data":{"type":"appStoreVersions","id":"VERSION_OTHER"}}}},{"type":"reviewSubmissionItems","id":"ITEM_TARGET","relationships":{"appStoreVersion":{"data":{"type":"appStoreVersions","id":"VERSION_123"}}}}],"links":{}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_123",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepAttachBuild:   true,
			stepSubmitReview:  true,
		},
	}
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if !checkpoint.Completed[stepSubmitReview] {
		t.Fatal("expected submit_review completion to survive when a later item holds the resumed version")
	}
}

// TestVerifyResumedCheckpointBindingKeepsLegacySubmissionCheckpoint proves a
// checkpoint recorded from the legacy appStoreVersionSubmissions flow is
// verified through the legacy per-version endpoint instead of being discarded
// on the modern endpoint's 404 every resume.
func TestVerifyResumedCheckpointBindingKeepsLegacySubmissionCheckpoint(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/LEGACY_SUB_123":
			return releaseJSONResponse(http.StatusNotFound, `{"errors":[{"status":"404","code":"NOT_FOUND","title":"Not Found"}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/appStoreVersionSubmission":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersionSubmissions","id":"LEGACY_SUB_123"}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "LEGACY_SUB_123",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepAttachBuild:   true,
			stepSubmitReview:  true,
		},
	}
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if !checkpoint.Completed[stepSubmitReview] {
		t.Fatal("expected legacy submission checkpoint to survive verification via the legacy endpoint")
	}
	if checkpoint.SubmissionID != "LEGACY_SUB_123" {
		t.Fatalf("expected legacy submission ID to survive, got %q", checkpoint.SubmissionID)
	}
}

// TestVerifyResumedCheckpointBindingAbortsOnIndeterminateSubmissionRead proves
// that a transient submission read failure aborts the resume instead of
// discarding the completion: re-running submit_review is not idempotent and
// could create a second submission.
func TestVerifyResumedCheckpointBindingAbortsOnIndeterminateSubmissionRead(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_123":
			return releaseJSONResponse(http.StatusInternalServerError, `{"errors":[{"status":"500","title":"Server error"}]}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_123",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepAttachBuild:   true,
			stepSubmitReview:  true,
		},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil)
	if err == nil {
		t.Fatal("expected an indeterminate submission read to abort the resume")
	}
	if !strings.Contains(err.Error(), "SUBMISSION_123") {
		t.Fatalf("expected error naming the submission, got %v", err)
	}
	if !checkpoint.Completed[stepSubmitReview] {
		t.Fatal("expected the completion to be preserved when verification is indeterminate")
	}
	if checkpoint.SubmissionID != "SUBMISSION_123" {
		t.Fatalf("expected submission ID to be preserved, got %q", checkpoint.SubmissionID)
	}
}

// TestVerifyResumedCheckpointBindingDropsSubmissionThatNoLongerExists pins the
// definitive contradiction: a 404 for the recorded submission discards the
// completion so the step runs again.
func TestVerifyResumedCheckpointBindingDropsSubmissionThatNoLongerExists(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_123":
			return releaseJSONResponse(http.StatusNotFound, `{"errors":[{"status":"404","code":"NOT_FOUND","title":"Not Found"}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/appStoreVersionSubmission":
			return releaseJSONResponse(http.StatusNotFound, `{"errors":[{"status":"404","code":"NOT_FOUND","title":"Not Found"}]}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_123",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepAttachBuild:   true,
			stepSubmitReview:  true,
		},
	}
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if checkpoint.Completed[stepSubmitReview] {
		t.Fatal("expected the completion for a missing submission to be discarded")
	}
	if checkpoint.SubmissionID != "" {
		t.Fatalf("expected the missing submission ID to be cleared, got %q", checkpoint.SubmissionID)
	}
}

// TestVerifyResumedCheckpointBindingDropsUnsubmittedDraftSubmission proves
// that a completed submit_review flag is discarded when the referenced
// submission is still a READY_FOR_REVIEW draft: it is bound to the selected
// version, but its state contradicts the claim that submission occurred.
func TestVerifyResumedCheckpointBindingDropsUnsubmittedDraftSubmission(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"reviewSubmissions","id":"SUBMISSION_123","attributes":{"state":"READY_FOR_REVIEW","platform":"IOS"},"relationships":{"appStoreVersionForReview":{"data":{"type":"appStoreVersions","id":"VERSION_123"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_123",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepAttachBuild:   true,
			stepSubmitReview:  true,
		},
	}
	var messages []string
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, func(message string) {
		messages = append(messages, message)
	}); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if checkpoint.Completed[stepSubmitReview] {
		t.Fatal("expected draft submit_review completion to be discarded")
	}
	if checkpoint.SubmissionID != "" {
		t.Fatalf("expected draft submission ID to be cleared, got %q", checkpoint.SubmissionID)
	}
	if !checkpoint.Completed[stepAttachBuild] {
		t.Fatal("expected proven attach_build completion to survive")
	}
	if len(messages) == 0 || !strings.Contains(strings.Join(messages, "\n"), "READY_FOR_REVIEW") {
		t.Fatalf("expected a diagnostic naming the contradicting state, got %v", messages)
	}
}

func TestVerifyResumedCheckpointBindingKeepsProvenCheckpoint(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"reviewSubmissions","id":"SUBMISSION_123","attributes":{"state":"WAITING_FOR_REVIEW","platform":"IOS"},"relationships":{"appStoreVersionForReview":{"data":{"type":"appStoreVersions","id":"VERSION_123"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_123",
		Completed: map[string]bool{
			stepEnsureVersion:     true,
			stepApplyMetadata:     true,
			stepAttachBuild:       true,
			stepValidateReadiness: true,
			stepSubmitReview:      true,
		},
	}
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if len(checkpoint.Completed) != 3 {
		t.Fatalf("expected only remotely verifiable completions to survive, got %#v", checkpoint.Completed)
	}
	if checkpoint.Completed[stepApplyMetadata] || checkpoint.Completed[stepValidateReadiness] {
		t.Fatalf("expected local unprovable completions to be discarded, got %#v", checkpoint.Completed)
	}
	if checkpoint.SubmissionID != "SUBMISSION_123" {
		t.Fatalf("expected verified submission ID to survive, got %q", checkpoint.SubmissionID)
	}
}

// TestExecuteRun_RejectsForgedCheckpointVersionBeforeMutation proves a modified
// checkpoint cannot substitute VersionID and have the pipeline act on it.
func TestExecuteRun_RejectsForgedCheckpointVersionBeforeMutation(t *testing.T) {
	origClientFactory := releaseClientFactory
	origMetadataExecutor := metadataPushExecutor
	origReadinessBuilder := readinessReportBuilder
	t.Cleanup(func() {
		releaseClientFactory = origClientFactory
		metadataPushExecutor = origMetadataExecutor
		readinessReportBuilder = origReadinessBuilder
	})

	metadataPushExecutor = func(context.Context, metadata.PushExecutionOptions) (metadata.PushPlanResult, error) {
		t.Fatal("metadata executor must not run for an unverifiable checkpoint")
		return metadata.PushPlanResult{}, nil
	}
	readinessReportBuilder = func(context.Context, validatecli.ReadinessOptions) (validation.Report, error) {
		t.Fatal("readiness builder must not run for an unverifiable checkpoint")
		return validation.Report{}, nil
	}

	var mutations []string
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			mutations = append(mutations, req.Method+" "+req.URL.Path)
		}
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_FORGED":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_FORGED","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_OTHER"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})
	releaseClientFactory = func() (*asc.Client, error) { return client, nil }

	checkpointPath := filepath.Join(t.TempDir(), "release-checkpoint.json")
	if err := saveCheckpoint(checkpointPath, runCheckpoint{
		AppID:       "APP_123",
		Version:     "2.4.0",
		BuildID:     "BUILD_123",
		MetadataDir: "./metadata/version/2.4.0",
		Platform:    "IOS",
		Mode:        releaseModeRun,
		VersionID:   "VERSION_FORGED",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepApplyMetadata: true,
			stepAttachBuild:   true,
		},
	}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	result, err := executeRun(context.Background(), runOptions{
		AppID:          "APP_123",
		Version:        "2.4.0",
		BuildID:        "BUILD_123",
		MetadataDir:    "./metadata/version/2.4.0",
		Platform:       "IOS",
		Timeout:        releaseRunTimeout,
		Confirm:        true,
		CheckpointFile: checkpointPath,
	})
	if err == nil {
		t.Fatal("expected executeRun to fail for a checkpoint version owned by another app")
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %q", result.Status)
	}
	if len(mutations) != 0 {
		t.Fatalf("expected no mutating requests, got %v", mutations)
	}
}

// TestVerifyResumedCheckpointBindingDropsReadinessWithIncompletePrerequisite
// proves an unsigned checkpoint cannot claim validate_readiness while a
// prerequisite mutation step is missing. The pipeline would run that mutation
// and then skip readiness, submitting a version whose readiness was never
// validated against the state the mutation produced.
func TestVerifyResumedCheckpointBindingDropsReadinessWithIncompletePrerequisite(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{
			stepEnsureVersion:     true,
			stepApplyMetadata:     true,
			stepValidateReadiness: true,
		},
	}
	var messages []string
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, func(message string) {
		messages = append(messages, message)
	}); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if checkpoint.Completed[stepValidateReadiness] {
		t.Fatal("expected validate_readiness to be discarded while attach_build is incomplete")
	}
	joined := strings.Join(messages, "\n")
	if !strings.Contains(joined, stepValidateReadiness) || !strings.Contains(joined, stepApplyMetadata) {
		t.Fatalf("expected diagnostics naming the unprovable local steps, got %v", messages)
	}
}

// TestVerifyResumedCheckpointBindingRerunsUnprovableLocalSteps proves an
// unsigned checkpoint cannot suppress operations whose effects cannot be
// authenticated from current App Store Connect state. Metadata may have
// changed locally after the checkpoint was written, and readiness is a
// point-in-time validation, so both steps must run on every resume.
func TestVerifyResumedCheckpointBindingRerunsUnprovableLocalSteps(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{
			stepEnsureVersion:     true,
			stepApplyMetadata:     true,
			stepAttachBuild:       true,
			stepValidateReadiness: true,
		},
	}
	var messages []string
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, func(message string) {
		messages = append(messages, message)
	}); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if checkpoint.Completed[stepApplyMetadata] {
		t.Fatal("expected unprovable apply_metadata completion to be discarded")
	}
	if checkpoint.Completed[stepValidateReadiness] {
		t.Fatal("expected point-in-time validate_readiness completion to be discarded")
	}
	if !checkpoint.Completed[stepEnsureVersion] || !checkpoint.Completed[stepAttachBuild] {
		t.Fatalf("expected authenticated remote-state completions to survive, got %#v", checkpoint.Completed)
	}
	joined := strings.Join(messages, "\n")
	if !strings.Contains(joined, stepApplyMetadata) || !strings.Contains(joined, stepValidateReadiness) {
		t.Fatalf("expected diagnostics naming both rerun steps, got %v", messages)
	}
}

// TestExecuteStage_PersistsDiscardedCompletionsBeforeMutating proves discarded
// completions reach the checkpoint file before the pipeline mutates anything.
// Otherwise a checkpoint write that fails after a successful re-attachment
// leaves the stale validate_readiness flag on disk, and the next resume — where
// the attachment now matches --build — skips readiness for the new build.
func TestExecuteStage_PersistsDiscardedCompletionsBeforeMutating(t *testing.T) {
	origClientFactory := releaseClientFactory
	origMetadataExecutor := metadataPushExecutor
	origReadinessBuilder := readinessReportBuilder
	t.Cleanup(func() {
		releaseClientFactory = origClientFactory
		metadataPushExecutor = origMetadataExecutor
		readinessReportBuilder = origReadinessBuilder
	})

	metadataPushExecutor = func(context.Context, metadata.PushExecutionOptions) (metadata.PushPlanResult, error) {
		return metadata.PushPlanResult{}, nil
	}
	readinessReportBuilder = func(context.Context, validatecli.ReadinessOptions) (validation.Report, error) {
		return validation.Report{}, nil
	}

	checkpointPath := filepath.Join(t.TempDir(), "release-checkpoint.json")
	var checkpointAtFirstMutation *runCheckpoint
	attached := false
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet && checkpointAtFirstMutation == nil {
			persisted, loadErr := loadCheckpoint(checkpointPath)
			if loadErr != nil {
				return nil, fmt.Errorf("load checkpoint during mutation: %w", loadErr)
			}
			checkpointAtFirstMutation = persisted
		}
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			if attached {
				return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
			}
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_OTHER","attributes":{"version":"41"}}}`)
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/appStoreVersions/VERSION_123/relationships/build":
			attached = true
			return releaseJSONResponse(http.StatusNoContent, ``)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})
	releaseClientFactory = func() (*asc.Client, error) { return client, nil }

	if err := saveCheckpoint(checkpointPath, runCheckpoint{
		AppID:       "APP_123",
		Version:     "2.4.0",
		BuildID:     "BUILD_123",
		MetadataDir: "./metadata/version/2.4.0",
		Platform:    "IOS",
		Mode:        releaseModeStage,
		VersionID:   "VERSION_123",
		Completed: map[string]bool{
			stepEnsureVersion:     true,
			stepApplyMetadata:     true,
			stepAttachBuild:       true,
			stepValidateReadiness: true,
		},
	}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	if _, err := executeStage(context.Background(), runOptions{
		AppID:          "APP_123",
		Version:        "2.4.0",
		BuildID:        "BUILD_123",
		MetadataDir:    "./metadata/version/2.4.0",
		Platform:       "IOS",
		Timeout:        releaseRunTimeout,
		Confirm:        true,
		CheckpointFile: checkpointPath,
	}); err != nil {
		t.Fatalf("executeStage error: %v", err)
	}

	if checkpointAtFirstMutation == nil {
		t.Fatal("expected the pipeline to re-attach the build")
	}
	if checkpointAtFirstMutation.Completed[stepAttachBuild] {
		t.Fatal("expected the discarded attach_build flag to be persisted before the re-attachment")
	}
	if checkpointAtFirstMutation.Completed[stepValidateReadiness] {
		t.Fatal("expected the discarded validate_readiness flag to be persisted before the re-attachment")
	}
}

// TestExecuteStageClearsAndPersistsUnprovenSubmissionID proves an unsigned
// stage checkpoint cannot inject a submission ID into structured output. Stage
// never submits for review, so a submission ID without a remotely verified
// submit_review completion is not trusted state.
func TestExecuteStageClearsAndPersistsUnprovenSubmissionID(t *testing.T) {
	origClientFactory := releaseClientFactory
	origMetadataExecutor := metadataPushExecutor
	origReadinessBuilder := readinessReportBuilder
	t.Cleanup(func() {
		releaseClientFactory = origClientFactory
		metadataPushExecutor = origMetadataExecutor
		readinessReportBuilder = origReadinessBuilder
	})

	metadataPushExecutor = func(context.Context, metadata.PushExecutionOptions) (metadata.PushPlanResult, error) {
		return metadata.PushPlanResult{}, nil
	}
	readinessReportBuilder = func(context.Context, validatecli.ReadinessOptions) (validation.Report, error) {
		return validation.Report{}, nil
	}

	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/APP_123/appStoreVersions":
			return releaseJSONResponse(http.StatusOK, `{"data":[{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"}}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})
	releaseClientFactory = func() (*asc.Client, error) { return client, nil }

	checkpointPath := filepath.Join(t.TempDir(), "release-checkpoint.json")
	if err := saveCheckpoint(checkpointPath, runCheckpoint{
		AppID:        "APP_123",
		Version:      "2.4.0",
		BuildID:      "BUILD_123",
		MetadataDir:  "./metadata/version/2.4.0",
		Platform:     "IOS",
		Mode:         releaseModeStage,
		SubmissionID: "FORGED_SUBMISSION",
		Completed:    map[string]bool{},
	}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	result, err := executeStage(context.Background(), runOptions{
		AppID:          "APP_123",
		Version:        "2.4.0",
		BuildID:        "BUILD_123",
		MetadataDir:    "./metadata/version/2.4.0",
		Platform:       "IOS",
		Timeout:        releaseRunTimeout,
		Confirm:        true,
		CheckpointFile: checkpointPath,
	})
	if err != nil {
		t.Fatalf("executeStage error: %v", err)
	}
	if result.SubmissionID != "" {
		t.Fatalf("stage output trusted an unproven submission ID: %q", result.SubmissionID)
	}

	persisted, err := loadCheckpoint(checkpointPath)
	if err != nil {
		t.Fatalf("load persisted checkpoint: %v", err)
	}
	if persisted == nil {
		t.Fatal("expected persisted checkpoint")
	}
	if persisted.SubmissionID != "" {
		t.Fatalf("persisted checkpoint retained unproven submission ID: %q", persisted.SubmissionID)
	}
}
