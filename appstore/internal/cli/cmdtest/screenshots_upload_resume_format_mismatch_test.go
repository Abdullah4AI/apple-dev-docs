package cmdtest

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

func TestRunScreenshotsUploadResumeRejectsFormatExtensionMismatchBeforeAnyRequest(t *testing.T) {
	tests := []struct {
		name        string
		displayType string
	}{
		// The pending file was swapped for JPEG bytes after the original
		// upload failed, which is what the resume must catch.
		{name: "artifact with display type", displayType: "APP_IPHONE_65"},
		// The check reads the bytes and the file name only, so an artifact
		// written before the display type was recorded is rejected too.
		{name: "artifact without display type"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			dir := t.TempDir()
			pendingPath := filepath.Join(dir, "01-home.png")
			// The original upload reserved a real PNG; the file on disk was
			// replaced with JPEG bytes under the same name before the resume.
			writeCmdtestScreenshotPNG(t, dir, "01-home.png")
			writeCmdtestScreenshotJPEG(t, dir, "01-home.png")

			artifact := map[string]any{
				"versionLocalizationId": "LOC_123",
				"rootPath":              dir,
				"setId":                 "set-1",
				"pendingFiles":          []string{pendingPath},
				"generatedAt":           "2026-08-19T00:00:00Z",
			}
			if test.displayType != "" {
				artifact["displayType"] = test.displayType
			}
			payload, err := json.Marshal(artifact)
			if err != nil {
				t.Fatalf("marshal artifact: %v", err)
			}
			artifactPath := filepath.Join(dir, "artifact.json")
			if err := os.WriteFile(artifactPath, payload, 0o600); err != nil {
				t.Fatalf("write artifact: %v", err)
			}

			var requests atomic.Int32
			originalTransport := http.DefaultTransport
			t.Cleanup(func() {
				http.DefaultTransport = originalTransport
			})
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests.Add(1)
				t.Errorf("unexpected request before resume preflight: %s %s", req.Method, req.URL.String())
				return nil, errors.New("unexpected request")
			})

			_, stderr := captureOutput(t, func() {
				if code := cmd.Run([]string{
					"screenshots", "upload",
					"--resume", artifactPath,
					"--output", "json",
				}, "1.2.3"); code == cmd.ExitSuccess {
					t.Fatal("expected resume to fail for JPEG data named .png")
				}
			})

			if got := requests.Load(); got != 0 {
				t.Fatalf("expected zero HTTP requests, got %d", got)
			}
			for _, want := range []string{"01-home.png", "JPEG", "01-home.jpg", "PNG"} {
				if !strings.Contains(stderr, want) {
					t.Fatalf("expected stderr to mention %q, got %q", want, stderr)
				}
			}
		})
	}
}
