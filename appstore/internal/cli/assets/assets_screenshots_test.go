package assets

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestAssetsScreenshotsSizesCommandFilter(t *testing.T) {
	cmd := AssetsScreenshotsSizesCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{"--display-type", "APP_IPHONE_65"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, stderr := captureOutput(t, func() {
		if err := cmd.Exec(context.Background(), cmd.FlagSet.Args()); err != nil {
			t.Fatalf("exec error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result asc.ScreenshotSizesResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(result.Sizes) != 1 {
		t.Fatalf("expected 1 size entry, got %d", len(result.Sizes))
	}
	if result.Sizes[0].DisplayType != "APP_IPHONE_65" {
		t.Fatalf("expected APP_IPHONE_65, got %q", result.Sizes[0].DisplayType)
	}
}

func TestAssetsScreenshotsSizesCommandSupportsIPhone69Alias(t *testing.T) {
	cmd := AssetsScreenshotsSizesCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{"--display-type", "IPHONE_69"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, stderr := captureOutput(t, func() {
		if err := cmd.Exec(context.Background(), cmd.FlagSet.Args()); err != nil {
			t.Fatalf("exec error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result asc.ScreenshotSizesResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(result.Sizes) != 1 {
		t.Fatalf("expected 1 size entry, got %d", len(result.Sizes))
	}
	if result.Sizes[0].DisplayType != "APP_IPHONE_69" {
		t.Fatalf("expected APP_IPHONE_69, got %q", result.Sizes[0].DisplayType)
	}
}

func TestAssetsScreenshotsSizesCommandSupportsIMessageIPhone69Alias(t *testing.T) {
	cmd := AssetsScreenshotsSizesCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{"--display-type", "IMESSAGE_APP_IPHONE_69"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, stderr := captureOutput(t, func() {
		if err := cmd.Exec(context.Background(), cmd.FlagSet.Args()); err != nil {
			t.Fatalf("exec error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result asc.ScreenshotSizesResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(result.Sizes) != 1 {
		t.Fatalf("expected 1 size entry, got %d", len(result.Sizes))
	}
	if result.Sizes[0].DisplayType != "IMESSAGE_APP_IPHONE_69" {
		t.Fatalf("expected IMESSAGE_APP_IPHONE_69, got %q", result.Sizes[0].DisplayType)
	}
}

func TestNormalizeScreenshotDisplayTypeAliasIPhone69Variants(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{input: "IPHONE_69", want: "APP_IPHONE_69"},
		{input: "APP_IPHONE_69", want: "APP_IPHONE_69"},
		{input: "imessage_app_iphone_69", want: "IMESSAGE_APP_IPHONE_69"},
	}

	for _, tc := range testCases {
		got, err := normalizeScreenshotDisplayType(tc.input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("expected %q for %q, got %q", tc.want, tc.input, got)
		}
	}
}

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	origStdout := os.Stdout
	origStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	_ = rOut.Close()
	_ = rErr.Close()

	return string(outBytes), string(errBytes)
}
