package storekit

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	storekitapi "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/storekit"
)

func TestReadPNGValidatesDocumentedImageRequirements(t *testing.T) {
	valid := writeTestPNG(t, image.Rect(0, 0, 3840, 160), color.NRGBA{R: 20, G: 30, B: 40, A: 255})
	if _, err := readPNG(valid, storekitapi.ImageSizeFull); err != nil {
		t.Fatalf("readPNG(valid) error = %v", err)
	}

	wrongSize := writeTestPNG(t, image.Rect(0, 0, 100, 100), color.NRGBA{A: 255})
	if _, err := readPNG(wrongSize, storekitapi.ImageSizeFull); err == nil || !strings.Contains(err.Error(), "3840 pixels wide") {
		t.Fatalf("readPNG(wrong size) error = %v", err)
	}

	transparent := writeTestPNG(t, image.Rect(0, 0, 1024, 1024), color.NRGBA{R: 20, A: 100})
	if _, err := readPNG(transparent, storekitapi.ImageSizeBulletPoint); err == nil || !strings.Contains(err.Error(), "must not contain transparency") {
		t.Fatalf("readPNG(transparent) error = %v", err)
	}
}

func TestPerformanceWaitFailsWhenAppleReportsFail(t *testing.T) {
	failed := &storekitapi.PerformanceTestResult{RequestID: "request-1", Result: "FAIL"}
	if err := performanceResultError(failed); err == nil || !strings.Contains(err.Error(), "request-1") {
		t.Fatalf("performanceResultError(FAIL) = %v", err)
	}
	passed := &storekitapi.PerformanceTestResult{RequestID: "request-2", Result: "PASS"}
	if err := performanceResultError(passed); err != nil {
		t.Fatalf("performanceResultError(PASS) = %v", err)
	}
}

func writeTestPNG(t *testing.T, bounds image.Rectangle, fill color.NRGBA) string {
	t.Helper()
	img := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.SetNRGBA(x, y, fill)
		}
	}
	path := filepath.Join(t.TempDir(), "image.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
