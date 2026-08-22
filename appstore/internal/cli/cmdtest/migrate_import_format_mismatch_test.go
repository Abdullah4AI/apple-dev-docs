package cmdtest

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestMigrateImportRejectsFormatExtensionMismatchBeforeAnyMutation(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	fastlaneDir := filepath.Join(t.TempDir(), "fastlane")
	metadataDir := filepath.Join(fastlaneDir, "metadata", "en-US")
	if err := os.MkdirAll(metadataDir, 0o755); err != nil {
		t.Fatalf("mkdir metadata: %v", err)
	}
	writeFile(t, filepath.Join(metadataDir, "description.txt"), "English description")

	screenshotsDir := filepath.Join(fastlaneDir, "screenshots", "en-US")
	if err := os.MkdirAll(screenshotsDir, 0o755); err != nil {
		t.Fatalf("mkdir screenshots: %v", err)
	}
	writeCmdtestScreenshotJPEG(t, screenshotsDir, "iphone_65_screen.png")

	var requests atomic.Int32
	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})
	// Discovery runs before the client exists, so a mismatch must not create
	// the localization or the screenshot set first.
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		t.Errorf("unexpected request before import discovery finished: %s %s", req.Method, req.URL.String())
		return nil, errors.New("unexpected request")
	})

	stdout, _, runErr := runMigrateImportWithOptions(t, fastlaneDir)

	if runErr == nil {
		t.Fatal("expected migrate import to fail for JPEG data named .png")
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("expected zero requests, got %d", got)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, want := range []string{"iphone_65_screen.png", "JPEG", "iphone_65_screen.jpg", "PNG"} {
		if !strings.Contains(runErr.Error(), want) {
			t.Fatalf("expected error to mention %q, got %v", want, runErr)
		}
	}
}
