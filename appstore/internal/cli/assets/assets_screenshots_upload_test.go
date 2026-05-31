package assets

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestUploadScreenshotsSkipExistingStartsUploadTimeoutAfterChecksumFiltering(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "200ms")
	t.Setenv("ASC_UPLOAD_TIMEOUT", "30s")

	filePath := writeAssetsTestPNG(t, t.TempDir(), "01-home.png")
	fileSizeBytes := fileSize(t, filePath)

	origChecksumFunc := screenshotFileChecksumFunc
	screenshotFileChecksumFunc = func(path string) (string, error) {
		time.Sleep(250 * time.Millisecond)
		return computeFileChecksum(path)
	}
	t.Cleanup(func() {
		screenshotFileChecksumFunc = origChecksumFunc
	})

	origTransport := http.DefaultTransport
	http.DefaultTransport = assetsUploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}

		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/LOC_123/appScreenshotSets":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshotSets","id":"set-1","attributes":{"screenshotDisplayType":"APP_IPHONE_65"}}],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appScreenshotSets/set-1/appScreenshots":
			return assetsJSONResponse(http.StatusOK, `{"data":[],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appScreenshotSets/set-1/relationships/appScreenshots":
			return assetsJSONResponse(http.StatusOK, `{"data":[],"links":{}}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/appScreenshots":
			body := fmt.Sprintf(`{"data":{"type":"appScreenshots","id":"new-1","attributes":{"uploadOperations":[{"method":"PUT","url":"https://upload.example/new-1","length":%d,"offset":0}]}}}`, fileSizeBytes)
			return assetsJSONResponse(http.StatusCreated, body)
		case req.Method == http.MethodPut && req.URL.Host == "upload.example":
			return assetsJSONResponse(http.StatusOK, `{}`)
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/appScreenshots/new-1":
			return assetsJSONResponse(http.StatusOK, `{"data":{"type":"appScreenshots","id":"new-1","attributes":{"uploaded":true}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appScreenshots/new-1":
			return assetsJSONResponse(http.StatusOK, `{"data":{"type":"appScreenshots","id":"new-1","attributes":{"assetDeliveryState":{"state":"COMPLETE"}}}}`)
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/appScreenshotSets/set-1/relationships/appScreenshots":
			return assetsJSONResponse(http.StatusNoContent, "")
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	client := newAssetsUploadTestClient(t)
	result, err := uploadScreenshots(context.Background(), client, "LOC_123", "APP_IPHONE_65", []string{filePath}, true, false, false)
	if err != nil {
		t.Fatalf("uploadScreenshots() error: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 upload result, got %d", len(result.Results))
	}
	if result.Results[0].AssetID != "new-1" {
		t.Fatalf("expected uploaded asset ID new-1, got %#v", result.Results[0])
	}
}

func TestUploadScreenshotsDryRunReportsWouldUpload(t *testing.T) {
	filePath := writeAssetsTestPNG(t, t.TempDir(), "01-home.png")

	origTransport := http.DefaultTransport
	http.DefaultTransport = assetsUploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/LOC_123/appScreenshotSets":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshotSets","id":"set-1","attributes":{"screenshotDisplayType":"APP_IPHONE_65"}}],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appScreenshotSets/set-1/appScreenshots":
			return assetsJSONResponse(http.StatusOK, `{"data":[],"links":{}}`)
		default:
			t.Fatalf("unexpected request in dry-run: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	client := newAssetsUploadTestClient(t)
	result, err := uploadScreenshots(context.Background(), client, "LOC_123", "APP_IPHONE_65", []string{filePath}, false, false, true)
	if err != nil {
		t.Fatalf("uploadScreenshots() error: %v", err)
	}

	if !result.DryRun {
		t.Fatal("expected DryRun=true")
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].State != "would-upload" {
		t.Fatalf("expected state would-upload, got %q", result.Results[0].State)
	}
	if result.Results[0].AssetID != "" {
		t.Fatalf("expected empty asset ID in dry-run, got %q", result.Results[0].AssetID)
	}
}

func TestUploadScreenshotsDryRunDoesNotCreateSet(t *testing.T) {
	filePath := writeAssetsTestPNG(t, t.TempDir(), "01-home.png")

	origTransport := http.DefaultTransport
	http.DefaultTransport = assetsUploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/LOC_123/appScreenshotSets":
			return assetsJSONResponse(http.StatusOK, `{"data":[],"links":{}}`)
		default:
			t.Fatalf("dry-run must not issue mutating requests: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	client := newAssetsUploadTestClient(t)
	result, err := uploadScreenshots(context.Background(), client, "LOC_123", "APP_IPHONE_65", []string{filePath}, false, false, true)
	if err != nil {
		t.Fatalf("uploadScreenshots() error: %v", err)
	}

	if !result.DryRun {
		t.Fatal("expected DryRun=true")
	}
	if result.SetID != "" {
		t.Fatalf("expected empty set ID when no set exists, got %q", result.SetID)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].State != "would-upload" {
		t.Fatalf("expected state would-upload, got %q", result.Results[0].State)
	}
}

func TestUploadScreenshotsDryRunWithReplaceReportsWouldDelete(t *testing.T) {
	filePath := writeAssetsTestPNG(t, t.TempDir(), "01-home.png")

	origTransport := http.DefaultTransport
	http.DefaultTransport = assetsUploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/LOC_123/appScreenshotSets":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshotSets","id":"set-1","attributes":{"screenshotDisplayType":"APP_IPHONE_65"}}],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appScreenshotSets/set-1/appScreenshots":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"existing-1","attributes":{"fileName":"old.png","fileSize":100}}],"links":{}}`)
		default:
			t.Fatalf("unexpected request in dry-run --replace: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	client := newAssetsUploadTestClient(t)
	result, err := uploadScreenshots(context.Background(), client, "LOC_123", "APP_IPHONE_65", []string{filePath}, false, true, true)
	if err != nil {
		t.Fatalf("uploadScreenshots() error: %v", err)
	}

	if !result.DryRun {
		t.Fatal("expected DryRun=true")
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results (1 delete + 1 upload), got %d", len(result.Results))
	}
	if result.Results[0].State != "would-delete" {
		t.Fatalf("expected first result state would-delete, got %q", result.Results[0].State)
	}
	if result.Results[0].AssetID != "existing-1" {
		t.Fatalf("expected would-delete asset ID existing-1, got %q", result.Results[0].AssetID)
	}
	if result.Results[1].State != "would-upload" {
		t.Fatalf("expected second result state would-upload, got %q", result.Results[1].State)
	}
	if result.Uploaded != 0 {
		t.Fatalf("expected uploaded=0 for dry-run replace preview, got %d", result.Uploaded)
	}
}

func TestUploadScreenshotsDryRunWithSkipExistingReportsSkipped(t *testing.T) {
	filePath := writeAssetsTestPNG(t, t.TempDir(), "01-home.png")
	checksum, err := computeFileChecksum(filePath)
	if err != nil {
		t.Fatalf("compute checksum: %v", err)
	}

	origTransport := http.DefaultTransport
	http.DefaultTransport = assetsUploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/LOC_123/appScreenshotSets":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshotSets","id":"set-1","attributes":{"screenshotDisplayType":"APP_IPHONE_65"}}],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appScreenshotSets/set-1/appScreenshots":
			return assetsJSONResponse(http.StatusOK, fmt.Sprintf(`{"data":[{"type":"appScreenshots","id":"existing-1","attributes":{"fileName":"01-home.png","fileSize":100,"sourceFileChecksum":"%s"}}],"links":{}}`, checksum))
		default:
			t.Fatalf("unexpected request in dry-run --skip-existing: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	client := newAssetsUploadTestClient(t)
	result, err := uploadScreenshots(context.Background(), client, "LOC_123", "APP_IPHONE_65", []string{filePath}, true, false, true)
	if err != nil {
		t.Fatalf("uploadScreenshots() error: %v", err)
	}

	if !result.DryRun {
		t.Fatal("expected DryRun=true")
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].State != "skipped" {
		t.Fatalf("expected state skipped, got %q", result.Results[0].State)
	}
	if !result.Results[0].Skipped {
		t.Fatal("expected Skipped=true")
	}
}

func TestExecuteAppScreenshotUploadMaxScreenshotsAccountsForExistingRemoteScreenshots(t *testing.T) {
	dir := t.TempDir()
	files := make([]string, 0, appScreenshotSetMaxScreenshots)
	sizes := make(map[string]int64, appScreenshotSetMaxScreenshots)
	for i := 1; i <= appScreenshotSetMaxScreenshots; i++ {
		filePath := writeAssetsTestPNG(t, dir, fmt.Sprintf("%02d-home.png", i))
		files = append(files, filePath)
		sizes[filepath.Base(filePath)] = fileSize(t, filePath)
	}

	createCount := 0
	origTransport := http.DefaultTransport
	http.DefaultTransport = assetsUploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/LOC_123/appScreenshotSets":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshotSets","id":"set-1","attributes":{"screenshotDisplayType":"APP_IPHONE_65"}}],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appScreenshotSets/set-1/appScreenshots":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"old-1","attributes":{"fileName":"old.png","fileSize":100}}],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appScreenshotSets/set-1/relationships/appScreenshots":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"old-1"}],"links":{}}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/appScreenshots":
			createCount++
			if createCount > appScreenshotSetMaxScreenshots-1 {
				t.Fatalf("max-screenshots with one existing remote screenshot must upload at most 9 new files; got create %d", createCount)
			}
			name := fmt.Sprintf("%02d-home.png", createCount)
			return assetsJSONResponse(http.StatusCreated, fmt.Sprintf(`{"data":{"type":"appScreenshots","id":"new-%d","attributes":{"uploadOperations":[{"method":"PUT","url":"https://upload.example/new-%d","length":%d,"offset":0}]}}}`, createCount, createCount, sizes[name]))
		case req.Method == http.MethodPut && req.URL.Host == "upload.example":
			return assetsJSONResponse(http.StatusOK, `{}`)
		case req.Method == http.MethodPatch && strings.HasPrefix(req.URL.Path, "/v1/appScreenshots/"):
			id := strings.TrimPrefix(req.URL.Path, "/v1/appScreenshots/")
			return assetsJSONResponse(http.StatusOK, fmt.Sprintf(`{"data":{"type":"appScreenshots","id":"%s","attributes":{"uploaded":true}}}`, id))
		case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/v1/appScreenshots/"):
			id := strings.TrimPrefix(req.URL.Path, "/v1/appScreenshots/")
			return assetsJSONResponse(http.StatusOK, fmt.Sprintf(`{"data":{"type":"appScreenshots","id":"%s","attributes":{"assetDeliveryState":{"state":"COMPLETE"}}}}`, id))
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/appScreenshotSets/set-1/relationships/appScreenshots":
			return assetsJSONResponse(http.StatusNoContent, "")
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	client := newAssetsUploadTestClient(t)
	result, err := executeAppScreenshotUpload(context.Background(), screenshotUploadConfig[asc.AppScreenshotUploadResult]{
		Client:         client,
		LocalizationID: "LOC_123",
		DisplayType:    "APP_IPHONE_65",
		Files:          files,
		MaxScreenshots: appScreenshotSetMaxScreenshots,
		Access:         appStoreVersionScreenshotSetAccess,
		BuildResult: func(localizationID string, set asc.Resource[asc.AppScreenshotSetAttributes], dryRun bool, results []asc.AssetUploadResultItem) asc.AppScreenshotUploadResult {
			return buildAppScreenshotUploadResult(localizationID, set, dryRun, results)
		},
	}, "")
	if err != nil {
		t.Fatalf("executeAppScreenshotUpload() error: %v", err)
	}
	if createCount != appScreenshotSetMaxScreenshots-1 {
		t.Fatalf("expected 9 uploaded screenshots, got %d", createCount)
	}
	if result.Uploaded != appScreenshotSetMaxScreenshots-1 {
		t.Fatalf("expected uploaded count 9, got %d", result.Uploaded)
	}
}

func TestExecuteAppScreenshotUploadRejectsAppendAboveScreenshotLimit(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		writeAssetsTestPNG(t, dir, "01-home.png"),
		writeAssetsTestPNG(t, dir, "02-home.png"),
	}

	origTransport := http.DefaultTransport
	http.DefaultTransport = assetsUploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/LOC_123/appScreenshotSets":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshotSets","id":"set-1","attributes":{"screenshotDisplayType":"APP_IPHONE_65"}}],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appScreenshotSets/set-1/appScreenshots":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"old-1"},{"type":"appScreenshots","id":"old-2"},{"type":"appScreenshots","id":"old-3"},{"type":"appScreenshots","id":"old-4"},{"type":"appScreenshots","id":"old-5"},{"type":"appScreenshots","id":"old-6"},{"type":"appScreenshots","id":"old-7"},{"type":"appScreenshots","id":"old-8"},{"type":"appScreenshots","id":"old-9"}],"links":{}}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/appScreenshots":
			t.Fatal("must reject before creating screenshots when append would exceed the set limit")
			return nil, nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	client := newAssetsUploadTestClient(t)
	_, err := executeAppScreenshotUpload(context.Background(), screenshotUploadConfig[asc.AppScreenshotUploadResult]{
		Client:         client,
		LocalizationID: "LOC_123",
		DisplayType:    "APP_IPHONE_65",
		Files:          files,
		Access:         appStoreVersionScreenshotSetAccess,
	}, "")
	if err == nil {
		t.Fatal("expected screenshot set limit error")
	}
	if !strings.Contains(err.Error(), "would exceed App Store screenshot set limit 10") {
		t.Fatalf("expected screenshot set limit error, got %v", err)
	}
}
