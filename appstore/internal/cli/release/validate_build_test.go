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

// TestExecuteStageRejectsBuildOwnedByAnotherAppBeforeMutating proves --build is
// checked before the pipeline mutates anything. attach_build runs after the
// metadata and routing coverage steps, so an unusable build used to be
// discovered only once those mutations had already been applied.
func TestExecuteStageRejectsBuildOwnedByAnotherAppBeforeMutating(t *testing.T) {
	origClientFactory := releaseClientFactory
	origMetadataExecutor := metadataPushExecutor
	origReadinessBuilder := readinessReportBuilder
	origTransport := http.DefaultTransport
	t.Cleanup(func() {
		releaseClientFactory = origClientFactory
		metadataPushExecutor = origMetadataExecutor
		readinessReportBuilder = origReadinessBuilder
		http.DefaultTransport = origTransport
	})

	metadataPushExecutor = func(context.Context, metadata.PushExecutionOptions) (metadata.PushPlanResult, error) {
		t.Fatal("metadata must not be applied for a build that belongs to another app")
		return metadata.PushPlanResult{}, nil
	}
	readinessReportBuilder = func(context.Context, validatecli.ReadinessOptions) (validation.Report, error) {
		t.Fatal("readiness must not run for a build that belongs to another app")
		return validation.Report{}, nil
	}

	var mutations []string
	http.DefaultTransport = releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			mutations = append(mutations, req.Method+" "+req.URL.Path)
		}
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/builds/BUILD_OTHER/relationships/app":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"apps","id":"APP_OTHER"}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/APP_123/appStoreVersions":
			return releaseJSONResponse(http.StatusOK, `{"data":[{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"}}]}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})
	testClient := newReleaseTestClient(t)
	releaseClientFactory = func() (*asc.Client, error) { return testClient, nil }

	result, err := executeStage(context.Background(), runOptions{
		AppID:          "APP_123",
		Version:        "2.4.0",
		BuildID:        "BUILD_OTHER",
		MetadataDir:    "./metadata/version/2.4.0",
		Platform:       "IOS",
		Timeout:        releaseRunTimeout,
		Confirm:        true,
		CheckpointFile: filepath.Join(t.TempDir(), "stage-checkpoint.json"),
	})
	if err == nil {
		t.Fatal("executeStage() error = nil, want a rejected build")
	}
	if !strings.Contains(err.Error(), "BUILD_OTHER") || !strings.Contains(err.Error(), "APP_OTHER") {
		t.Fatalf("expected an error naming the build and its app, got %v", err)
	}
	if result.FailedStep != stepValidateBuild {
		t.Fatalf("expected the %s step to fail, got %q", stepValidateBuild, result.FailedStep)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected the pipeline to stop at the build check, got %#v", result.Steps)
	}
	if len(mutations) != 0 {
		t.Fatalf("expected no mutating requests, got %v", mutations)
	}
}

// TestExecuteStageDryRunRejectsUnknownBuild proves the preview fails on a build
// that does not exist instead of reporting a clean plan for it.
func TestExecuteStageDryRunRejectsUnknownBuild(t *testing.T) {
	origClientFactory := releaseClientFactory
	origMetadataExecutor := metadataPushExecutor
	origTransport := http.DefaultTransport
	t.Cleanup(func() {
		releaseClientFactory = origClientFactory
		metadataPushExecutor = origMetadataExecutor
		http.DefaultTransport = origTransport
	})

	metadataPushExecutor = func(context.Context, metadata.PushExecutionOptions) (metadata.PushPlanResult, error) {
		t.Fatal("metadata must not be planned for an unknown build")
		return metadata.PushPlanResult{}, nil
	}

	http.DefaultTransport = releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && req.URL.Path == "/v1/builds/BUILD_TYPO/relationships/app" {
			return releaseJSONResponse(http.StatusNotFound, `{"errors":[{"status":"404","code":"NOT_FOUND","title":"Not Found"}]}`)
		}
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
	})
	testClient := newReleaseTestClient(t)
	releaseClientFactory = func() (*asc.Client, error) { return testClient, nil }

	result, err := executeStage(context.Background(), runOptions{
		AppID:          "APP_123",
		Version:        "2.4.0",
		BuildID:        "BUILD_TYPO",
		MetadataDir:    "./metadata/version/2.4.0",
		Platform:       "IOS",
		Timeout:        releaseRunTimeout,
		DryRun:         true,
		CheckpointFile: filepath.Join(t.TempDir(), "stage-checkpoint.json"),
	})
	if err == nil {
		t.Fatal("executeStage() error = nil, want an unusable build")
	}
	if !strings.Contains(err.Error(), "BUILD_TYPO") {
		t.Fatalf("expected an error naming the build, got %v", err)
	}
	if result.Status != "error" || result.FailedStep != stepValidateBuild {
		t.Fatalf("expected the dry-run to fail at the build check, got status %q step %q", result.Status, result.FailedStep)
	}
}

// TestExecuteStageValidatesBuildOnceBeforeAttaching proves the precondition read
// resolves the build the operator asked for and does not replace the
// attachment check that follows it.
func TestExecuteStageValidatesBuildOnceBeforeAttaching(t *testing.T) {
	origClientFactory := releaseClientFactory
	origMetadataExecutor := metadataPushExecutor
	origReadinessBuilder := readinessReportBuilder
	origTransport := http.DefaultTransport
	t.Cleanup(func() {
		releaseClientFactory = origClientFactory
		metadataPushExecutor = origMetadataExecutor
		readinessReportBuilder = origReadinessBuilder
		http.DefaultTransport = origTransport
	})

	metadataApplied := false
	metadataPushExecutor = func(context.Context, metadata.PushExecutionOptions) (metadata.PushPlanResult, error) {
		metadataApplied = true
		return metadata.PushPlanResult{VersionID: "VERSION_123"}, nil
	}
	readinessReportBuilder = func(context.Context, validatecli.ReadinessOptions) (validation.Report, error) {
		return validation.Report{Summary: validation.Summary{}}, nil
	}

	buildLookups := 0
	requestOrder := make([]string, 0, 4)
	http.DefaultTransport = releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestOrder = append(requestOrder, req.Method+" "+req.URL.Path)
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/builds/BUILD_123/relationships/app":
			buildLookups++
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"apps","id":"APP_123"}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/APP_123/appStoreVersions":
			return releaseJSONResponse(http.StatusOK, `{"data":[{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"}}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})
	testClient := newReleaseTestClient(t)
	releaseClientFactory = func() (*asc.Client, error) { return testClient, nil }

	result, err := executeStage(context.Background(), runOptions{
		AppID:          "APP_123",
		Version:        "2.4.0",
		BuildID:        "BUILD_123",
		MetadataDir:    "./metadata/version/2.4.0",
		Platform:       "IOS",
		Timeout:        releaseRunTimeout,
		Confirm:        true,
		CheckpointFile: filepath.Join(t.TempDir(), "stage-checkpoint.json"),
	})
	if err != nil {
		t.Fatalf("executeStage error: %v", err)
	}
	if !metadataApplied {
		t.Fatal("expected the pipeline to continue past the build check")
	}
	if buildLookups != 1 {
		t.Fatalf("expected exactly one build precondition read, got %d (%v)", buildLookups, requestOrder)
	}
	if len(result.Steps) != 5 || result.Steps[0].Name != stepValidateBuild {
		t.Fatalf("expected the build check to be reported first, got %#v", result.Steps)
	}
	if requestOrder[0] != "GET /v1/builds/BUILD_123/relationships/app" {
		t.Fatalf("expected the build to be resolved before anything else, got %v", requestOrder)
	}
}
