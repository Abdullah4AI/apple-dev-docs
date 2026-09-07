package asc

import (
	"strings"
	"testing"
)

func TestPrintTableWebAppDistributionSetReceiptUsesRegistry(t *testing.T) {
	result := &WebAppDistributionSetResult{
		AppID:                 "app-123",
		DistributionType:      "CUSTOM",
		EducationDiscountType: "NOT_APPLICABLE",
		Changed:               true,
		Verified:              true,
		Status:                "verified",
	}
	for _, render := range []struct {
		name string
		fn   func(any) error
	}{
		{name: "table", fn: PrintTable},
		{name: "markdown", fn: PrintMarkdown},
	} {
		t.Run(render.name, func(t *testing.T) {
			output := captureStdout(t, func() error { return render.fn(result) })
			for _, want := range []string{"App ID", "Distribution Type", "Education Discount Type", "Changed", "Verified", "Status", "app-123", "CUSTOM", "NOT_APPLICABLE", "true", "verified"} {
				if !strings.Contains(output, want) {
					t.Fatalf("output missing %q: %q", want, output)
				}
			}
		})
	}
}
