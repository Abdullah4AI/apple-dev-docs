package cmdtest

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func writeBuildUploadReconciliationIPA(t *testing.T, path string) int64 {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create IPA fixture: %v", err)
	}
	archive := zip.NewWriter(file)
	plist, err := archive.Create("Payload/Demo.app/Info.plist")
	if err != nil {
		_ = file.Close()
		t.Fatalf("create Info.plist entry: %v", err)
	}
	const infoPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleIdentifier</key>
  <string>com.example.demo</string>
  <key>CFBundleShortVersionString</key>
  <string>1.0.0</string>
  <key>CFBundleVersion</key>
  <string>42</string>
</dict>
</plist>`
	if _, err := io.WriteString(plist, infoPlist); err != nil {
		_ = archive.Close()
		_ = file.Close()
		t.Fatalf("write Info.plist entry: %v", err)
	}
	if err := archive.Close(); err != nil {
		_ = file.Close()
		t.Fatalf("close IPA archive: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close IPA fixture: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat IPA fixture: %v", err)
	}
	return info.Size()
}

func TestBuildsUploadRecoversAmbiguousCommitFromBuildUploadState(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	ipaPath := filepath.Join(t.TempDir(), "app.ipa")
	ipaSize := writeBuildUploadReconciliationIPA(t, ipaPath)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	reconciliationChecks := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/123456789":
			return jsonResponse(http.StatusOK, `{"data":{"type":"apps","id":"123456789","attributes":{"name":"Demo","bundleId":"com.example.demo"}}}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/buildUploads":
			return jsonResponse(http.StatusOK, `{"data":{"type":"buildUploads","id":"upload-1","attributes":{"cfBundleShortVersionString":"1.0.0","cfBundleVersion":"42","platform":"IOS"}}}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/buildUploadFiles":
			body := fmt.Sprintf(`{"data":{"type":"buildUploadFiles","id":"file-1","attributes":{"fileName":"app.ipa","fileSize":%d,"uti":"com.apple.itunes.ipa","assetType":"ASSET","uploadOperations":[{"method":"PUT","url":"https://upload.example.com/part-1","length":%d,"offset":0,"requestHeaders":[{"name":"Content-Type","value":"application/octet-stream"}]}]}}}`, ipaSize, ipaSize)
			return jsonResponse(http.StatusOK, body)
		case req.Method == http.MethodPut && req.URL.Host == "upload.example.com":
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Errorf("read uploaded IPA: %v", err)
				return nil, fmt.Errorf("read uploaded IPA: %w", err)
			}
			if int64(len(body)) != ipaSize {
				err := fmt.Errorf("expected %d uploaded bytes, got %d", ipaSize, len(body))
				t.Error(err)
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/buildUploadFiles/file-1":
			return jsonResponse(http.StatusServiceUnavailable, `{"errors":[{"status":"503","code":"SERVICE_UNAVAILABLE","title":"Service unavailable"}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/buildUploads/upload-1":
			reconciliationChecks++
			return jsonResponse(http.StatusOK, `{"data":{"type":"buildUploads","id":"upload-1","attributes":{"state":{"state":"PROCESSING"}}}}`)
		default:
			err := fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
			t.Error(err)
			return nil, err
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"builds", "upload",
			"--app", "123456789",
			"--ipa", ipaPath,
			"--version", "1.0.0",
			"--build-number", "42",
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if runErr != nil {
		t.Fatalf("builds upload error: %v", runErr)
	}
	if reconciliationChecks != 1 {
		t.Fatalf("expected one reconciliation lookup, got %d", reconciliationChecks)
	}
	if !strings.Contains(stderr, "Upload committed in App Store Connect.") {
		t.Fatalf("expected commit progress on stderr, got %q", stderr)
	}
	var output asc.BuildUploadResult
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode output: %v\nstdout=%s", err, stdout)
	}
	if output.UploadID != "upload-1" || output.Uploaded == nil || !*output.Uploaded {
		t.Fatalf("expected reconciled upload output, got %#v", output)
	}
}
