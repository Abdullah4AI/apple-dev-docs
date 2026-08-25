package cmdtest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

func TestVideoPreviewsUploadReportsUploadedPreviewsWhenALaterFileFails(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	pathDir := t.TempDir()
	firstSize := writePreviewFile(t, filepath.Join(pathDir, "01-first.mov"))
	secondSize := writePreviewFile(t, filepath.Join(pathDir, "02-second.mov"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/LOC_123/appPreviewSets":
			return statusJSONResponse(`{"data":[{"type":"appPreviewSets","id":"set-1","attributes":{"previewType":"IPHONE_65"}}],"links":{}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appPreviewSets/set-1/appPreviews":
			return statusJSONResponse(`{"data":[],"links":{}}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/appPreviews":
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read create preview body: %v", err)
			}
			if strings.Contains(string(body), "02-second.mov") {
				return statusJSONResponse(fmt.Sprintf(
					`{"data":{"type":"appPreviews","id":"preview-second","attributes":{"fileName":"02-second.mov","assetDeliveryState":{"state":"AWAITING_UPLOAD"},"uploadOperations":[{"method":"PUT","url":"https://upload.example/preview-second","length":%d,"offset":0}]}}}`,
					secondSize,
				)), nil
			}
			return statusJSONResponse(fmt.Sprintf(
				`{"data":{"type":"appPreviews","id":"preview-first","attributes":{"fileName":"01-first.mov","uploadOperations":[{"method":"PUT","url":"https://upload.example/preview-first","length":%d,"offset":0}]}}}`,
				firstSize,
			)), nil
		case req.Method == http.MethodPut && req.URL.Host == "upload.example" && req.URL.Path == "/preview-second":
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("preview upload failed")),
				Header:     http.Header{},
			}, nil
		case req.Method == http.MethodPut && req.URL.Host == "upload.example":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/appPreviews/preview-first":
			return statusJSONResponse(`{"data":{"type":"appPreviews","id":"preview-first","attributes":{"uploaded":true}}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appPreviews/preview-first":
			return statusJSONResponse(`{"data":{"type":"appPreviews","id":"preview-first","attributes":{"assetDeliveryState":{"state":"COMPLETE"}}}}`), nil
		case req.Method == http.MethodDelete && (req.URL.Path == "/v1/appPreviews/preview-first" || req.URL.Path == "/v1/appPreviews/preview-second"):
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	exitCode := rootcmd.ExitSuccess
	stdout, stderr := captureOutput(t, func() {
		exitCode = rootcmd.Run([]string{
			"video-previews", "upload",
			"--version-localization", "LOC_123",
			"--path", pathDir,
			"--device-type", "IPHONE_65",
			"--output", "json",
		}, "1.2.3")
	})

	if exitCode == rootcmd.ExitSuccess {
		t.Fatal("expected a non-zero exit code when a preview upload fails")
	}
	if !strings.Contains(stderr, "video-previews upload") {
		t.Fatalf("expected the failure to be attributed to video-previews upload, got stderr=%q", stderr)
	}

	var payload struct {
		SetID   string `json:"setId"`
		Results []struct {
			FileName string `json:"fileName"`
			AssetID  string `json:"assetId"`
			State    string `json:"state"`
		} `json:"results"`
		Failures []struct {
			FileName string `json:"fileName"`
			FilePath string `json:"filePath"`
			Error    string `json:"error"`
		} `json:"failures"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode partial receipt: %v\nstdout=%s", err, stdout)
	}
	if payload.SetID != "set-1" {
		t.Fatalf("setId = %q, want set-1", payload.SetID)
	}

	uploaded := make(map[string]string, len(payload.Results))
	for _, item := range payload.Results {
		uploaded[item.FileName] = item.AssetID
	}
	if uploaded["01-first.mov"] != "preview-first" {
		t.Fatalf("expected the already uploaded preview in the receipt, got %s", stdout)
	}

	if len(payload.Failures) != 1 {
		t.Fatalf("expected exactly one failure entry, got %s", stdout)
	}
	failure := payload.Failures[0]
	if failure.FileName != "02-second.mov" {
		t.Fatalf("failure fileName = %q, want 02-second.mov", failure.FileName)
	}
	if !strings.Contains(failure.FilePath, "02-second.mov") {
		t.Fatalf("failure filePath = %q, want the failing preview path", failure.FilePath)
	}
	if strings.TrimSpace(failure.Error) == "" {
		t.Fatalf("expected a failure message, got %s", stdout)
	}

	rolledBackStates := 0
	for _, item := range payload.Results {
		if (item.AssetID == "preview-first" || item.AssetID == "preview-second") && item.State == "rolled-back" {
			rolledBackStates++
		}
	}
	if rolledBackStates != 2 {
		t.Fatalf("expected both receipt items to be marked rolled-back after cleanup, got %s", stdout)
	}
}
