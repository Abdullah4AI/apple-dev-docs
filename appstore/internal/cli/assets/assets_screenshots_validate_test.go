package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateScreenshotAssetsSortsEntriesAndKeepsHiddenWarningsNonBlocking(t *testing.T) {
	dir := t.TempDir()
	writeAssetsTestPNGWithSize(t, dir, "02-details.png", 1242, 2688)
	writeAssetsTestPNGWithSize(t, dir, "01-home.png", 1242, 2688)
	writeAssetsTestPNGWithSize(t, dir, ".hidden.png", 1242, 2688)

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
	writeAssetsTestPNGWithSize(t, dir, "02-details.png", 1242, 2688)
	writeAssetsTestPNGWithSize(t, dir, "01-home.png", 1242, 2688)
	writeAssetsTestPNGWithSize(t, dir, ".hidden.png", 1242, 2688)

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
