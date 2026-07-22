//go:build !darwin

package xcode

import (
	"context"
	"strings"
	"testing"
)

func TestGenerateManualExportOptions_IsUnsupportedOutsideMacOS(t *testing.T) {
	_, err := generateManualExportOptions(context.Background(), "Demo.xcarchive", "TEAM123")
	if err == nil || !strings.Contains(err.Error(), "only supported on macOS") {
		t.Fatalf("expected clear unsupported-platform error, got %v", err)
	}
}
