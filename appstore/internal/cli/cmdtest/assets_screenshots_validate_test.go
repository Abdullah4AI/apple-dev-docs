package cmdtest

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type screenshotValidateOutput struct {
	Path         string                          `json:"path"`
	DisplayType  string                          `json:"displayType"`
	TotalFiles   int                             `json:"totalFiles"`
	ErrorCount   int                             `json:"errorCount"`
	WarningCount int                             `json:"warningCount"`
	Files        []screenshotValidateFileOutput  `json:"files"`
	Issues       []screenshotValidateIssueOutput `json:"issues"`
}

type screenshotValidateFileOutput struct {
	Order    int    `json:"order"`
	FileName string `json:"fileName"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

type screenshotValidateIssueOutput struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	FileName string `json:"fileName,omitempty"`
	Message  string `json:"message"`
}

func TestScreenshotsHelpListsValidateSubcommand(t *testing.T) {
	stdout, stderr, runErr := runRootCommand(t, []string{"screenshots"})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "\n  validate") {
		t.Fatalf("expected screenshots help to list validate, got %q", stderr)
	}
}

func TestAssetsScreenshotsValidateOutputReportsOrderingHiddenFilesAndDimensionErrors(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "02-details.png"), 1242, 2688)
	writePNG(t, filepath.Join(dir, "01-home.png"), 1242, 2688)
	writePNG(t, filepath.Join(dir, ".hidden.png"), 1242, 2688)
	writePNG(t, filepath.Join(dir, "03-bad.png"), 100, 100)

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"screenshots", "validate",
			"--path", dir,
			"--device-type", "IPHONE_65",
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(t.Context())
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if runErr == nil {
		t.Fatal("expected reported validation error, got nil")
	}
	if !strings.Contains(runErr.Error(), "screenshots validate: found 1 error(s)") {
		t.Fatalf("expected reported error count, got %v", runErr)
	}

	var result screenshotValidateOutput
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}

	if result.Path != dir {
		t.Fatalf("expected path %q, got %q", dir, result.Path)
	}
	if result.DisplayType != "APP_IPHONE_65" {
		t.Fatalf("expected normalized display type APP_IPHONE_65, got %q", result.DisplayType)
	}
	if result.TotalFiles != 4 {
		t.Fatalf("expected 4 discovered files, got %d", result.TotalFiles)
	}
	if result.ErrorCount != 1 {
		t.Fatalf("expected 1 error, got %d", result.ErrorCount)
	}
	if result.WarningCount != 1 {
		t.Fatalf("expected 1 warning, got %d", result.WarningCount)
	}
	if len(result.Files) != 4 {
		t.Fatalf("expected 4 file rows, got %d", len(result.Files))
	}

	wantOrder := []string{".hidden.png", "01-home.png", "02-details.png", "03-bad.png"}
	for i, want := range wantOrder {
		if result.Files[i].Order != i+1 {
			t.Fatalf("expected order %d for row %d, got %d", i+1, i, result.Files[i].Order)
		}
		if result.Files[i].FileName != want {
			t.Fatalf("expected file %q at row %d, got %q", want, i, result.Files[i].FileName)
		}
	}

	if !hasScreenshotValidateIssue(result.Issues, "hidden_file", "warning", ".hidden.png") {
		t.Fatalf("expected hidden-file warning in issues, got %+v", result.Issues)
	}
	if !hasScreenshotValidateIssue(result.Issues, "dimension_mismatch", "error", "03-bad.png") {
		t.Fatalf("expected dimension mismatch error in issues, got %+v", result.Issues)
	}
}

func hasScreenshotValidateIssue(issues []screenshotValidateIssueOutput, code, severity, fileName string) bool {
	for _, issue := range issues {
		if issue.Code == code && issue.Severity == severity && issue.FileName == fileName {
			return true
		}
	}
	return false
}

func TestAssetsScreenshotsValidateHelpUsesCanonicalUsage(t *testing.T) {
	usage := usageForCommand(t, "screenshots", "validate")
	if !strings.Contains(usage, `asc screenshots validate --path "./screenshots" --device-type "IPHONE_65"`) {
		t.Fatalf("expected canonical validate usage, got %q", usage)
	}
}

func TestAssetsScreenshotsValidateReportsUnreadableHiddenDotfiles(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "01-home.png"), 1242, 2688)
	if err := os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("not an image"), 0o644); err != nil {
		t.Fatalf("write dotfile: %v", err)
	}

	stdout, stderr, runErr := runRootCommand(t, []string{
		"screenshots", "validate",
		"--path", dir,
		"--device-type", "IPHONE_65",
		"--output", "json",
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if runErr == nil {
		t.Fatal("expected reported validation error, got nil")
	}

	var result screenshotValidateOutput
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}

	if !hasScreenshotValidateIssue(result.Issues, "hidden_file", "warning", ".DS_Store") {
		t.Fatalf("expected hidden dotfile warning, got %+v", result.Issues)
	}
	if !hasScreenshotValidateIssue(result.Issues, "read_failure", "error", ".DS_Store") {
		t.Fatalf("expected unreadable-dotfile error, got %+v", result.Issues)
	}
}

func TestAssetsScreenshotsValidateRejectsInvalidDeviceTypeAsUsageError(t *testing.T) {
	dir := t.TempDir()

	stdout, stderr, runErr := runRootCommand(t, []string{
		"screenshots", "validate",
		"--path", dir,
		"--device-type", "not-a-device",
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "unsupported screenshot display type") {
		t.Fatalf("expected invalid device-type error, got %q", stderr)
	}
}

func TestAssetsScreenshotsValidatePreservesRawPathSemanticsForFileWithTrailingSeparator(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shot.png")
	writePNG(t, path, 1242, 2688)

	stdout, stderr, runErr := runRootCommand(t, []string{
		"screenshots", "validate",
		"--path", path + string(filepath.Separator),
		"--device-type", "IPHONE_65",
		"--output", "json",
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if runErr == nil {
		t.Fatal("expected path error, got nil")
	}
	if !strings.Contains(runErr.Error(), "screenshots validate:") {
		t.Fatalf("expected validate command context, got %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", runErr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}
