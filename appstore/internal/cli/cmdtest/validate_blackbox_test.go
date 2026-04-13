package cmdtest

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildASCBlackBoxBinary(t *testing.T) string {
	t.Helper()

	repoRoot := findModuleRoot(t)
	binaryPath := filepath.Join(t.TempDir(), "asc")

	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/appledev")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("failed to build asc binary: %v\n%s", err, string(output))
	}

	return binaryPath
}

func findModuleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find module root from %s", dir)
		}
		dir = parent
	}
}

func TestValidateRemovedRemediationFlagsReturnUsageExitCode(t *testing.T) {
	binaryPath := buildASCBlackBoxBinary(t)

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "next removed",
			args:    []string{"validate", "--app", "app-1", "--version-id", "ver-1", "--next"},
			wantErr: "flag provided but not defined",
		},
		{
			name:    "fix-plan removed",
			args:    []string{"validate", "--app", "app-1", "--version-id", "ver-1", "--fix-plan"},
			wantErr: "flag provided but not defined",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, test.args...)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected process exit error, got %v", err)
			}
			if exitErr.ExitCode() != 2 {
				t.Fatalf("expected exit code 2, got %d", exitErr.ExitCode())
			}
			if stdout.String() != "" {
				t.Fatalf("expected empty stdout, got %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), test.wantErr) {
				t.Fatalf("expected error %q, got %q", test.wantErr, stderr.String())
			}
		})
	}
}

func TestValidateSubcommandsRejectParentValidateFlagsExitCode(t *testing.T) {
	binaryPath := buildASCBlackBoxBinary(t)

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "version-id before subcommand",
			args:    []string{"validate", "--version-id", "ver-1", "testflight", "--app", "app-1", "--build", "build-1"},
			wantErr: "--version-id is only valid for asc validate",
		},
		{
			name:    "strict before subcommand",
			args:    []string{"validate", "--strict", "testflight", "--app", "app-1", "--build", "build-1"},
			wantErr: "--strict must be passed after the validate subcommand name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, test.args...)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected process exit error, got %v", err)
			}
			if exitErr.ExitCode() != 2 {
				t.Fatalf("expected exit code 2, got %d", exitErr.ExitCode())
			}
			if stdout.String() != "" {
				t.Fatalf("expected empty stdout, got %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), test.wantErr) {
				t.Fatalf("expected error %q, got %q", test.wantErr, stderr.String())
			}
		})
	}
}
