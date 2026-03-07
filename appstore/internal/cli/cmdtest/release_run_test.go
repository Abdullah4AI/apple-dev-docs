package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestRelease_ShowsHelp(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"release"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if !strings.Contains(stderr, "release") {
		t.Fatalf("expected help to mention release command, got %q", stderr)
	}
}

func TestReleaseRun_MissingConfirm(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--app", "APP_123",
			"--version", "1.2.3",
			"--build", "BUILD_123",
			"--metadata-dir", "./metadata/version/1.2.3",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if !strings.Contains(stderr, "--confirm is required unless --dry-run is set") {
		t.Fatalf("expected missing confirm error, got %q", stderr)
	}
}

func TestReleaseRun_MissingMetadataDir(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--app", "APP_123",
			"--version", "1.2.3",
			"--build", "BUILD_123",
			"--dry-run",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if !strings.Contains(stderr, "--metadata-dir is required") {
		t.Fatalf("expected missing metadata-dir error, got %q", stderr)
	}
}

func TestReleaseRun_InvalidPlatform(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--app", "APP_123",
			"--version", "1.2.3",
			"--build", "BUILD_123",
			"--metadata-dir", "./metadata/version/1.2.3",
			"--platform", "BAD_PLATFORM",
			"--dry-run",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if !strings.Contains(stderr, "--platform must be one of") {
		t.Fatalf("expected unsupported platform error, got %q", stderr)
	}
}

func TestReleaseRun_InvalidTimeout(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--app", "APP_123",
			"--version", "1.2.3",
			"--build", "BUILD_123",
			"--metadata-dir", "./metadata/version/1.2.3",
			"--dry-run",
			"--timeout", "0s",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if !strings.Contains(stderr, "--timeout must be greater than 0") {
		t.Fatalf("expected invalid timeout error, got %q", stderr)
	}
}
