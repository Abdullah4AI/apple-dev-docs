package cmdtest

import (
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

func TestRunScreenshotsUploadRejectsFormatExtensionMismatchBeforeAnyRequest(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	workDir := t.TempDir()
	writeCmdtestScreenshotJPEG(t, workDir, "01-home.png")

	var requests atomic.Int32
	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		t.Errorf("unexpected request before upload preflight: %s %s", req.Method, req.URL.String())
		return nil, errors.New("unexpected request")
	})

	_, stderr := captureOutput(t, func() {
		code := cmd.Run([]string{
			"screenshots", "upload",
			"--version-localization", "LOC_123",
			"--path", workDir,
			"--device-type", "IPHONE_65",
			"--output", "json",
		}, "1.2.3")
		if code == cmd.ExitSuccess {
			t.Fatal("expected the upload to fail for JPEG data named .png")
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
}

func writeCmdtestScreenshotJPEG(t *testing.T, dir, name string) {
	t.Helper()

	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create jpeg: %v", err)
	}
	defer file.Close()

	img := image.NewRGBA(image.Rect(0, 0, 1242, 2688))
	for y := 0; y < 2688; y++ {
		for x := 0; x < 1242; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	if err := jpeg.Encode(file, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
}
