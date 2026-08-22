package cmdtest

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

func TestRunScreenshotsUploadFanoutRejectsFormatExtensionMismatchBeforeAnyRequest(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "plain fan-out"},
		// --replace deletes a locale's existing screenshots before uploading,
		// so the preflight must reject the file before the first request.
		{name: "replace fan-out", args: []string{"--replace", "--confirm"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			root := t.TempDir()
			localeDir := filepath.Join(root, "en-US")
			if err := os.MkdirAll(localeDir, 0o755); err != nil {
				t.Fatalf("create locale dir: %v", err)
			}
			writeCmdtestScreenshotJPEG(t, localeDir, "01-home.png")

			var requests atomic.Int32
			originalTransport := http.DefaultTransport
			t.Cleanup(func() {
				http.DefaultTransport = originalTransport
			})
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests.Add(1)
				t.Errorf("unexpected request before fan-out preflight: %s %s", req.Method, req.URL.String())
				return nil, errors.New("unexpected request")
			})

			args := append([]string{
				"screenshots", "upload",
				"--app", "123456789",
				"--version", "1.2.3",
				"--path", root,
				"--device-type", "IPHONE_65",
				"--output", "json",
			}, test.args...)

			_, stderr := captureOutput(t, func() {
				if code := cmd.Run(args, "1.2.3"); code == cmd.ExitSuccess {
					t.Fatal("expected fan-out upload to fail for JPEG data named .png")
				}
			})

			if got := requests.Load(); got != 0 {
				t.Fatalf("expected zero HTTP requests, got %d", got)
			}
			for _, want := range []string{"en-US", "01-home.png", "JPEG", "01-home.jpg", "PNG"} {
				if !strings.Contains(stderr, want) {
					t.Fatalf("expected stderr to mention %q, got %q", want, stderr)
				}
			}
		})
	}
}
