package cmdtest

import (
	"context"
	"io"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestMigrateMetadataPullAliasDelegatesToMetadataCommand(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"migrate", "metadata", "pull", "--version", "1.2.3", "--dir", "./metadata"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !shared.IsReportedUsageError(runErr) {
		t.Fatalf("expected reported usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "Error: --app is required (or set ASC_APP_ID)\n" {
		t.Fatalf("expected concise metadata pull validation error, got %q", stderr)
	}
}

func TestMigrateMetadataPushAliasDelegatesToMetadataCommand(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"migrate", "metadata", "push", "--version", "1.2.3", "--dir", "./metadata", "--dry-run"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !shared.IsReportedUsageError(runErr) {
		t.Fatalf("expected reported usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "Error: --app is required (or set ASC_APP_ID)\n" {
		t.Fatalf("expected concise metadata push validation error, got %q", stderr)
	}
}

func TestMigrateMetadataValidateAliasDelegatesToMetadataCommand(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"migrate", "metadata", "validate"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !shared.IsReportedUsageError(runErr) {
		t.Fatalf("expected reported usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if stderr != "Error: --dir is required\n" {
		t.Fatalf("expected concise metadata validate validation error, got %q", stderr)
	}
}
