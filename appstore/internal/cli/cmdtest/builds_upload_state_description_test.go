package cmdtest

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// App Store Connect returns state details as `code` plus `description`; it never
// sends `message`. The upload failure must surface Apple's human-readable text.
func TestBuildsUploadWaitSurfacesStateErrorDescriptions(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	restoreDiagnostics := shared.SetBuildUploadFailureDiagnosticsForTesting(func(context.Context, *asc.Client, string, *asc.BuildUploadResponse) (string, error) {
		return "", nil
	})
	t.Cleanup(restoreDiagnostics)

	ipaPath := writeBuildUploadIPA(t, "com.example.demo")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/123456789":
			return jsonResponse(http.StatusOK, `{"data":{"type":"apps","id":"123456789","attributes":{"name":"Demo","bundleId":"com.example.demo"}}}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/buildUploads":
			return jsonResponse(http.StatusOK, `{"data":{"type":"buildUploads","id":"upload-1","attributes":{"cfBundleShortVersionString":"1.0.0","cfBundleVersion":"42","platform":"IOS"}}}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/buildUploadFiles":
			return jsonResponse(http.StatusOK, `{"data":{"type":"buildUploadFiles","id":"file-1","attributes":{"fileName":"app.ipa","fileSize":4,"uti":"com.apple.itunes.ipa","assetType":"ASSET","uploadOperations":[{"method":"PUT","url":"https://upload.example.com/part-1","length":4,"offset":0,"requestHeaders":[{"name":"Content-Type","value":"application/octet-stream"}]}]}}}`)
		case req.Method == http.MethodPut && req.URL.Host == "upload.example.com":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/buildUploadFiles/file-1":
			return jsonResponse(http.StatusOK, `{"data":{"type":"buildUploadFiles","id":"file-1","attributes":{"uploaded":true}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/preReleaseVersions":
			return jsonResponse(http.StatusOK, `{"data":[]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/buildUploads/upload-1":
			return jsonResponse(http.StatusOK, `{"data":{"type":"buildUploads","id":"upload-1","attributes":{"cfBundleShortVersionString":"1.0.0","cfBundleVersion":"42","platform":"IOS","state":{"state":"FAILED","errors":[{"code":"ERROR_ITMS_90XXX","description":"Invalid Bundle. The bundle does not support the minimum OS version."}]}}}}`)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	captureOutput(t, func() {
		if err := root.Parse([]string{
			"builds", "upload",
			"--app", "123456789",
			"--ipa", ipaPath,
			"--version", "1.0.0",
			"--build-number", "42",
			"--wait",
			"--poll-interval", "1ms",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if runErr == nil {
		t.Fatal("expected builds upload to fail, got nil")
	}
	if !strings.Contains(runErr.Error(), "ERROR_ITMS_90XXX") {
		t.Fatalf("expected Apple error code in error, got %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "Invalid Bundle. The bundle does not support the minimum OS version.") {
		t.Fatalf("expected Apple error description in error, got %v", runErr)
	}
}
