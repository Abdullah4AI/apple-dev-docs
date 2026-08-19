package builds

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestBuildsMissingRequiredInputExposesStructuredDiagnostic(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	var err error
	stderr := captureBuildsDiagnosticStderr(t, func() {
		err = BuildsUploadCommand().ParseAndRun(context.Background(), nil)
	})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v, want flag.ErrHelp contract", err)
	}
	if got, want := err.Error(), "--app"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
	if want := "Error: --app is required (or set ASC_APP_ID)\n"; !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want diagnostic %q", stderr, want)
	}

	diagnostic, ok := shared.DiagnosticFromError(err)
	if !ok {
		t.Fatalf("DiagnosticFromError(%v) did not find metadata", err)
	}
	if diagnostic.Code != shared.DiagnosticRequiredInputMissing || diagnostic.Parameter != "--app" {
		t.Fatalf("diagnostic = %+v, want required_input_missing for --app", diagnostic)
	}
}

func TestBuildsMissingArtifactLeavesDiagnosticParameterEmpty(t *testing.T) {
	var err error
	stderr := captureBuildsDiagnosticStderr(t, func() {
		err = BuildsUploadCommand().ParseAndRun(context.Background(), []string{"--app", "123456789"})
	})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v, want flag.ErrHelp contract", err)
	}
	if want := "Error: --ipa or --pkg is required\n"; !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want diagnostic %q", stderr, want)
	}

	diagnostic, ok := shared.DiagnosticFromError(err)
	if !ok {
		t.Fatalf("DiagnosticFromError(%v) did not find metadata", err)
	}
	if diagnostic.Code != shared.DiagnosticRequiredInputMissing || diagnostic.Parameter != "" {
		t.Fatalf("diagnostic = %+v, want required_input_missing without a parameter", diagnostic)
	}
}

func TestBuildsMissingExpireSelectorLeavesDiagnosticParameterEmpty(t *testing.T) {
	var err error
	stderr := captureBuildsDiagnosticStderr(t, func() {
		err = BuildsExpireAllCommand().ParseAndRun(context.Background(), []string{"--app", "123456789"})
	})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v, want flag.ErrHelp contract", err)
	}
	if want := "Error: --older-than or --keep-latest is required\n"; !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want diagnostic %q", stderr, want)
	}

	diagnostic, ok := shared.DiagnosticFromError(err)
	if !ok {
		t.Fatalf("DiagnosticFromError(%v) did not find metadata", err)
	}
	if diagnostic.Code != shared.DiagnosticRequiredInputMissing || diagnostic.Parameter != "" {
		t.Fatalf("diagnostic = %+v, want required_input_missing without a parameter", diagnostic)
	}
}

func captureBuildsDiagnosticStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stderr = writer
	defer func() {
		os.Stderr = original
		_ = reader.Close()
		_ = writer.Close()
	}()

	fn()
	_ = writer.Close()
	os.Stderr = original
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	return string(data)
}
