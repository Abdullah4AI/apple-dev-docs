package assets

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestAssetsPreviewsUploadCommandRejectsSkipExistingWithReplace(t *testing.T) {
	cmd := AssetsPreviewsUploadCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{
		"--version-localization", "LOC_ID",
		"--path", "./previews",
		"--device-type", "IPHONE_65",
		"--skip-existing",
		"--replace",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		runErr = cmd.Exec(context.Background(), cmd.FlagSet.Args())
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "--skip-existing and --replace are mutually exclusive") {
		t.Fatalf("expected mutually exclusive error in stderr, got %q", stderr)
	}
}

func TestAssetsPreviewsUploadCommandRejectsUnsupportedFileBeforeAuth(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a-preview.mov", "b-poster.png"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("not-empty"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	clientCalled := false
	cmd := assetsPreviewsUploadCommandWithDependencies(previewUploadDependencies{
		GetClient: func() (*asc.Client, error) {
			clientCalled = true
			return &asc.Client{}, nil
		},
	})
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{
		"--version-localization", "LOC_ID",
		"--path", dir,
		"--device-type", "IPHONE_65",
		"--replace",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var runErr error
	stdout, _ := captureOutput(t, func() {
		runErr = cmd.Exec(context.Background(), cmd.FlagSet.Args())
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if runErr == nil {
		t.Fatal("expected unsupported preview file error")
	}
	if !strings.Contains(runErr.Error(), `unsupported preview file extension ".png"`) {
		t.Fatalf("expected local file-type error before auth, got %v", runErr)
	}
	if clientCalled {
		t.Fatal("expected preview file validation before auth/client creation")
	}
}

func TestDetectPreviewMimeTypeRejectsRegisteredNonVideoMIME(t *testing.T) {
	_, err := detectPreviewMimeType("poster.png")
	if err == nil {
		t.Fatal("expected unsupported preview file error")
	}
	if !strings.Contains(err.Error(), `unsupported preview file extension ".png"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectPreviewMimeTypeUsesSupportedVideoMapping(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "preview.mov", want: "video/quicktime"},
		{path: "preview.m4v", want: "video/x-m4v"},
		{path: "preview.MP4", want: "video/mp4"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := detectPreviewMimeType(tt.path)
			if err != nil {
				t.Fatalf("detectPreviewMimeType(%q) error: %v", tt.path, err)
			}
			if got != tt.want {
				t.Fatalf("detectPreviewMimeType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestUploadPreviewsRejectsUnsupportedFileBeforeRequests(t *testing.T) {
	tests := []struct {
		name    string
		replace bool
		dryRun  bool
	}{
		{name: "replace", replace: true},
		{name: "dry run", dryRun: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(t.TempDir(), "poster.png")
			if err := os.WriteFile(filePath, []byte("not-empty"), 0o600); err != nil {
				t.Fatalf("write invalid preview: %v", err)
			}

			requests := 0
			deleteRequests := 0
			client := newAssetsUploadTestServerClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method == http.MethodDelete {
					deleteRequests++
				}
				writeAssetsTestJSON(w, http.StatusInternalServerError, `{"errors":[{"status":"500","detail":"unexpected request"}]}`)
			}))

			_, err := uploadPreviews(
				context.Background(),
				client,
				"LOC_ID",
				"IPHONE_65",
				[]string{filePath},
				false,
				tt.replace,
				tt.dryRun,
			)
			if err == nil {
				t.Fatal("expected unsupported preview file error")
			}
			if !strings.Contains(err.Error(), `unsupported preview file extension ".png"`) {
				t.Fatalf("unexpected error: %v", err)
			}
			if requests != 0 {
				t.Fatalf("requests = %d, want 0", requests)
			}
			if deleteRequests != 0 {
				t.Fatalf("DELETE requests = %d, want 0", deleteRequests)
			}
		})
	}
}

func TestNormalizePreviewTypeCanonicalizesIPhone69Alias(t *testing.T) {
	testCases := []string{
		"IPHONE_69",
		"APP_IPHONE_69",
		" app_iphone_69 ",
	}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			got, err := normalizePreviewType(input)
			if err != nil {
				t.Fatalf("normalizePreviewType(%q) error: %v", input, err)
			}
			if got != "IPHONE_67" {
				t.Fatalf("normalizePreviewType(%q) = %q, want %q", input, got, "IPHONE_67")
			}
		})
	}
}

func TestNormalizePreviewTypeRejectsUnknownType(t *testing.T) {
	_, err := normalizePreviewType("IPHONE_70")
	if err == nil {
		t.Fatal("normalizePreviewType() expected an error")
	}
	if !strings.Contains(err.Error(), `unsupported preview type "IPHONE_70"`) {
		t.Fatalf("normalizePreviewType() error = %q", err)
	}
}

func TestIsValidPreviewFrameTimeCode(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "frame format", value: "00:00:05:00", want: true},
		{name: "millisecond format", value: "00:00:05.000", want: true},
		{name: "frame upper bound", value: "99:59:59:29", want: true},
		{name: "non numeric", value: "abc", want: false},
		{name: "missing component", value: "00:00:05", want: false},
		{name: "invalid minute", value: "00:60:05:00", want: false},
		{name: "invalid second", value: "00:00:60.000", want: false},
		{name: "invalid frame", value: "00:00:05:30", want: false},
		{name: "invalid millisecond width", value: "00:00:05.00", want: false},
		{name: "invalid separator", value: "00-00-05-00", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidPreviewFrameTimeCode(tt.value); got != tt.want {
				t.Fatalf("isValidPreviewFrameTimeCode(%q) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}
