package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/migrate"
)

const migrateDemoPasswordSentinel = "asc-red-sentinel-migrate-demo-pw-71ce40"

func setupMigrateReviewFixture(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	reviewDir := filepath.Join(root, "metadata", "review_information")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatalf("mkdir review_information: %v", err)
	}
	writeFile(t, filepath.Join(reviewDir, "first_name.txt"), "Rita")
	writeFile(t, filepath.Join(reviewDir, "email_address.txt"), "rita@example.com")
	writeFile(t, filepath.Join(reviewDir, "demo_user.txt"), "reviewer@example.com")
	writeFile(t, filepath.Join(reviewDir, "demo_password.txt"), migrateDemoPasswordSentinel)
	writeFile(t, filepath.Join(reviewDir, "demo_required.txt"), "true")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
}

func runMigrateImportDryRun(t *testing.T, extraArgs ...string) (string, string) {
	t.Helper()

	rootCmd := RootCommand("1.2.3")
	rootCmd.FlagSet.SetOutput(io.Discard)

	args := append([]string{
		"migrate", "import",
		"--app", "APP_ID",
		"--version-id", "VERSION_ID",
		"--dry-run",
	}, extraArgs...)

	return captureOutput(t, func() {
		if err := rootCmd.Parse(args); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := rootCmd.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
}

func TestMigrateImportDryRunRedactsDemoAccountPasswordInEveryFormat(t *testing.T) {
	formats := []struct {
		name string
		args []string
	}{
		{name: "json", args: []string{"--output", "json"}},
		{name: "pretty json", args: []string{"--output", "json", "--pretty"}},
		{name: "table", args: []string{"--output", "table"}},
		{name: "markdown", args: []string{"--output", "markdown"}},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			setupMigrateReviewFixture(t)

			stdout, stderr := runMigrateImportDryRun(t, format.args...)

			assertNoSentinel(t, "stdout", migrateDemoPasswordSentinel, stdout)
			assertNoSentinel(t, "stderr", migrateDemoPasswordSentinel, stderr)
			if !strings.Contains(stdout, redactedDemoPasswordText) {
				t.Fatalf("expected %q placeholder in output, got %q", redactedDemoPasswordText, stdout)
			}
			if !strings.Contains(stdout, "reviewer@example.com") {
				t.Fatalf("expected non-sensitive review fields to survive redaction, got %q", stdout)
			}
		})
	}
}

func TestMigrateImportDryRunIncludesDemoAccountPasswordWithIncludeSensitive(t *testing.T) {
	setupMigrateReviewFixture(t)

	stdout, stderr := runMigrateImportDryRun(t, "--output", "json", "--include-sensitive")

	var result migrate.MigrateImportResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("unmarshal result: %v\nstdout=%s", err, stdout)
	}
	if result.ReviewInformation == nil || result.ReviewInformation.DemoAccountPassword == nil {
		t.Fatal("expected review information with a demo account password")
	}
	if *result.ReviewInformation.DemoAccountPassword != migrateDemoPasswordSentinel {
		t.Fatalf("expected real password with --include-sensitive, got %q", *result.ReviewInformation.DemoAccountPassword)
	}
	if !strings.Contains(stderr, includeSensitiveWarningText) {
		t.Fatalf("expected plaintext-secret warning on stderr, got %q", stderr)
	}
}
