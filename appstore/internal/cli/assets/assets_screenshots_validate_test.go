package assets

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateScreenshotAssetsSortsEntriesAndKeepsHiddenWarningsNonBlocking(t *testing.T) {
	dir := t.TempDir()
	writeAssetsScreenshotValidatePNG(t, dir, "02-details.png", 1242, 2688, color.RGBA{R: 20, A: 255}, png.DefaultCompression)
	writeAssetsScreenshotValidatePNG(t, dir, "01-home.png", 1242, 2688, color.RGBA{R: 10, A: 255}, png.DefaultCompression)
	writeAssetsScreenshotValidatePNG(t, dir, ".hidden.png", 1242, 2688, color.RGBA{R: 30, A: 255}, png.DefaultCompression)

	result, err := validateScreenshotAssets(dir, "APP_IPHONE_65")
	if err != nil {
		t.Fatalf("validateScreenshotAssets() error: %v", err)
	}

	if result.ErrorCount != 0 {
		t.Fatalf("expected 0 errors, got %d", result.ErrorCount)
	}
	if result.WarningCount != 1 {
		t.Fatalf("expected 1 warning, got %d", result.WarningCount)
	}
	if result.ReadyFiles != 3 {
		t.Fatalf("expected 3 ready files, got %d", result.ReadyFiles)
	}

	wantOrder := []string{".hidden.png", "01-home.png", "02-details.png"}
	for i, want := range wantOrder {
		if result.Files[i].FileName != want {
			t.Fatalf("expected file %q at index %d, got %q", want, i, result.Files[i].FileName)
		}
		if result.Files[i].Order != i+1 {
			t.Fatalf("expected order %d at index %d, got %d", i+1, i, result.Files[i].Order)
		}
	}

	if !hasScreenshotValidateIssueWithSeverity(result.Issues, "hidden_file", screenshotValidateSeverityWarning, ".hidden.png") {
		t.Fatalf("expected hidden-file warning, got %+v", result.Issues)
	}
}

func TestValidateScreenshotAssetsMatchesUploadOrdering(t *testing.T) {
	dir := t.TempDir()
	writeAssetsScreenshotValidatePNG(t, dir, "02-details.png", 1242, 2688, color.RGBA{R: 20, A: 255}, png.DefaultCompression)
	writeAssetsScreenshotValidatePNG(t, dir, "01-home.png", 1242, 2688, color.RGBA{R: 10, A: 255}, png.DefaultCompression)
	writeAssetsScreenshotValidatePNG(t, dir, ".hidden.png", 1242, 2688, color.RGBA{R: 30, A: 255}, png.DefaultCompression)

	uploadFiles, err := collectAssetFiles(dir)
	if err != nil {
		t.Fatalf("collectAssetFiles() error: %v", err)
	}

	result, err := validateScreenshotAssets(dir, "APP_IPHONE_65")
	if err != nil {
		t.Fatalf("validateScreenshotAssets() error: %v", err)
	}

	if len(result.Files) != len(uploadFiles) {
		t.Fatalf("expected %d files, got %d", len(uploadFiles), len(result.Files))
	}
	for i, uploadFile := range uploadFiles {
		if result.Files[i].FilePath != uploadFile {
			t.Fatalf("expected validate path %q at index %d, got %q", uploadFile, i, result.Files[i].FilePath)
		}
		if result.Files[i].FileName != filepath.Base(uploadFile) {
			t.Fatalf("expected validate file name %q at index %d, got %q", filepath.Base(uploadFile), i, result.Files[i].FileName)
		}
	}
}

func TestValidateScreenshotAssetsReportsUnreadableDotfilesAndDimensionMismatch(t *testing.T) {
	dir := t.TempDir()
	writeAssetsTestPNGWithSize(t, dir, "01-home.png", 1242, 2688)
	writeAssetsTestPNGWithSize(t, dir, "03-bad.png", 100, 100)
	if err := os.WriteFile(dir+"/.DS_Store", []byte("not an image"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	result, err := validateScreenshotAssets(dir, "APP_IPHONE_65")
	if err != nil {
		t.Fatalf("validateScreenshotAssets() error: %v", err)
	}

	if result.ErrorCount != 2 {
		t.Fatalf("expected 2 errors, got %d", result.ErrorCount)
	}
	if result.WarningCount != 1 {
		t.Fatalf("expected 1 warning, got %d", result.WarningCount)
	}
	if result.ReadyFiles != 1 {
		t.Fatalf("expected 1 ready file, got %d", result.ReadyFiles)
	}

	if !hasScreenshotValidateIssueWithSeverity(result.Issues, "hidden_file", screenshotValidateSeverityWarning, ".DS_Store") {
		t.Fatalf("expected hidden-file warning, got %+v", result.Issues)
	}
	if !hasScreenshotValidateIssueWithSeverity(result.Issues, "read_failure", screenshotValidateSeverityError, ".DS_Store") {
		t.Fatalf("expected read-failure error, got %+v", result.Issues)
	}
	if !hasScreenshotValidateIssueWithSeverity(result.Issues, "dimension_mismatch", screenshotValidateSeverityError, "03-bad.png") {
		t.Fatalf("expected dimension mismatch error, got %+v", result.Issues)
	}
}

func TestValidateScreenshotAssetsReportsDecodedPixelDuplicatesDeterministically(t *testing.T) {
	dir := t.TempDir()
	marker := color.RGBA{R: 10, G: 20, B: 30, A: 255}
	originalPath := writeAssetsScreenshotValidatePNG(t, dir, "01-original.png", 312, 390, marker, png.BestSpeed)
	duplicatePath := writeAssetsScreenshotValidatePNG(t, dir, "02-duplicate.png", 312, 390, marker, png.BestCompression)
	writeAssetsScreenshotValidatePNG(t, dir, "03-another-duplicate.png", 312, 390, marker, png.DefaultCompression)
	writeAssetsScreenshotValidatePNG(t, dir, "04-distinct.png", 312, 390, color.RGBA{R: 11, G: 20, B: 30, A: 255}, png.DefaultCompression)

	originalBytes, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	duplicateBytes, err := os.ReadFile(duplicatePath)
	if err != nil {
		t.Fatalf("read duplicate: %v", err)
	}
	if bytes.Equal(originalBytes, duplicateBytes) {
		t.Fatal("expected differently encoded PNG files")
	}

	result, err := validateScreenshotAssets(dir, "APP_WATCH_SERIES_3")
	if err != nil {
		t.Fatalf("validateScreenshotAssets() error: %v", err)
	}

	if result.ErrorCount != 2 {
		t.Fatalf("expected 2 duplicate errors, got %d (%+v)", result.ErrorCount, result.Issues)
	}
	if result.ReadyFiles != 2 {
		t.Fatalf("expected 2 ready files, got %d", result.ReadyFiles)
	}
	if len(result.Files) != 4 {
		t.Fatalf("expected 4 files, got %d", len(result.Files))
	}
	if result.Files[0].Status != "ok" || result.Files[1].Status != "error" || result.Files[2].Status != "error" || result.Files[3].Status != "ok" {
		t.Fatalf("unexpected file statuses: %+v", result.Files)
	}

	for _, fileName := range []string{"02-duplicate.png", "03-another-duplicate.png"} {
		if !hasScreenshotValidateIssueWithSeverity(result.Issues, "duplicate_content", screenshotValidateSeverityError, fileName) {
			t.Fatalf("expected duplicate-content issue for %s, got %+v", fileName, result.Issues)
		}
	}
	for _, issue := range result.Issues {
		if issue.Code == "duplicate_content" && !strings.Contains(issue.Message, `"01-original.png"`) {
			t.Fatalf("expected deterministic original in duplicate message, got %q", issue.Message)
		}
	}
}

func TestValidateScreenshotAssetsPreserves16BitSamplesWhenDetectingDuplicates(t *testing.T) {
	dir := t.TempDir()
	writeAssetsScreenshotValidatePNG64(t, dir, "01-first.png", 312, 390, color.NRGBA64{R: 0x1201, G: 0x3400, B: 0x5600, A: 0xffff})
	writeAssetsScreenshotValidatePNG64(t, dir, "02-second.png", 312, 390, color.NRGBA64{R: 0x1202, G: 0x3400, B: 0x5600, A: 0xffff})

	result, err := validateScreenshotAssets(dir, "APP_WATCH_SERIES_3")
	if err != nil {
		t.Fatalf("validateScreenshotAssets() error: %v", err)
	}

	if result.ErrorCount != 0 {
		t.Fatalf("expected distinct 16-bit samples to pass, got %d error(s): %+v", result.ErrorCount, result.Issues)
	}
	if result.ReadyFiles != 2 {
		t.Fatalf("expected 2 ready files, got %d", result.ReadyFiles)
	}
}

func TestRenderScreenshotValidateResultSkipsRedundantAPIDisplayTypeRow(t *testing.T) {
	result := &screenshotValidateResult{
		Path:         "/tmp/screenshots",
		DisplayType:  "APP_IPHONE_65",
		TotalFiles:   1,
		ReadyFiles:   1,
		Files:        []screenshotValidateFile{{Order: 1, FilePath: "/tmp/screenshots/01-home.png", FileName: "01-home.png", Width: 1242, Height: 2688, Status: "ok"}},
		ErrorCount:   0,
		WarningCount: 0,
	}

	stdout, stderr := captureOutput(t, func() {
		if err := renderScreenshotValidateResult(result, false); err != nil {
			t.Fatalf("renderScreenshotValidateResult() error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.Contains(stdout, "apiDisplayType") {
		t.Fatalf("expected redundant apiDisplayType row to be omitted, got %q", stdout)
	}
}

func TestRenderScreenshotValidateResultIncludesCanonicalAPIDisplayTypeRowWhenItDiffers(t *testing.T) {
	result := &screenshotValidateResult{
		Path:           "/tmp/screenshots",
		DisplayType:    "APP_IPHONE_69",
		APIDisplayType: "APP_IPHONE_67",
		TotalFiles:     1,
		ReadyFiles:     1,
		Files:          []screenshotValidateFile{{Order: 1, FilePath: "/tmp/screenshots/01-home.png", FileName: "01-home.png", Width: 1290, Height: 2796, Status: "ok"}},
	}

	stdout, stderr := captureOutput(t, func() {
		if err := renderScreenshotValidateResult(result, false); err != nil {
			t.Fatalf("renderScreenshotValidateResult() error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "apiDisplayType") {
		t.Fatalf("expected canonical apiDisplayType row, got %q", stdout)
	}
	if !strings.Contains(stdout, "APP_IPHONE_67") {
		t.Fatalf("expected canonical API display type in output, got %q", stdout)
	}
}

func hasScreenshotValidateIssueWithSeverity(issues []screenshotValidateIssue, code, severity, fileName string) bool {
	for _, issue := range issues {
		if issue.Code == code && issue.Severity == severity && issue.FileName == fileName {
			return true
		}
	}
	return false
}

func writeAssetsScreenshotValidatePNG(t *testing.T, dir, name string, width, height int, marker color.RGBA, compression png.CompressionLevel) string {
	t.Helper()

	path := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.SetRGBA(0, 0, marker)

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer file.Close()

	encoder := png.Encoder{CompressionLevel: compression}
	if err := encoder.Encode(file, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return path
}

func writeAssetsScreenshotValidatePNG64(t *testing.T, dir, name string, width, height int, marker color.NRGBA64) {
	t.Helper()

	path := filepath.Join(dir, name)
	img := image.NewNRGBA64(image.Rect(0, 0, width, height))
	img.SetNRGBA64(0, 0, marker)

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}
