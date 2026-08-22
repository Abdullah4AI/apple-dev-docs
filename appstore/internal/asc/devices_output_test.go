package asc

import (
	"testing"
)

func TestDeviceBatchRegistrationSummaryRendererIncludesSummaryAndItems(t *testing.T) {
	summary := &DeviceBatchRegistrationSummary{
		InputFile:  "devices.txt",
		Total:      1,
		Processed:  1,
		Registered: 1,
		Results: []DeviceBatchRegistrationItem{{
			Row:      2,
			ID:       "device-1",
			Name:     "Test Device",
			UDID:     "UDID-1",
			Platform: "IOS",
			Status:   "registered",
		}},
	}

	renderCalls := 0
	err := renderByRegistry(summary, func(headers []string, rows [][]string) {
		renderCalls++
		if len(headers) == 0 || len(rows) != 1 {
			t.Fatalf("unexpected rendered section: headers=%v rows=%v", headers, rows)
		}
	})
	if err != nil {
		t.Fatalf("renderByRegistry() error: %v", err)
	}
	if renderCalls != 2 {
		t.Fatalf("expected summary and item tables, got %d sections", renderCalls)
	}
}
