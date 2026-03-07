package asc

import (
	"strings"
	"testing"
)

func TestPrintTable_AppStoreVersionPhasedReleaseIncludesProgress(t *testing.T) {
	resp := &AppStoreVersionPhasedReleaseResponse{
		Data: Resource[AppStoreVersionPhasedReleaseAttributes]{
			ID: "phase-1",
			Attributes: AppStoreVersionPhasedReleaseAttributes{
				PhasedReleaseState: PhasedReleaseStateActive,
				StartDate:          "2026-02-20",
				CurrentDayNumber:   3,
				TotalPauseDuration: 0,
			},
		},
	}

	output := captureStdout(t, func() error {
		return PrintTable(resp)
	})

	if !strings.Contains(output, "Progress") {
		t.Fatalf("expected progress header in output, got: %s", output)
	}
	if !strings.Contains(output, "[####------] 3/7") {
		t.Fatalf("expected progress bar in output, got: %s", output)
	}
}

func TestPrintMarkdown_AppStoreVersionPhasedReleaseIncludesProgress(t *testing.T) {
	resp := &AppStoreVersionPhasedReleaseResponse{
		Data: Resource[AppStoreVersionPhasedReleaseAttributes]{
			ID: "phase-1",
			Attributes: AppStoreVersionPhasedReleaseAttributes{
				PhasedReleaseState: PhasedReleaseStateActive,
				StartDate:          "2026-02-20",
				CurrentDayNumber:   3,
				TotalPauseDuration: 0,
			},
		},
	}

	output := captureStdout(t, func() error {
		return PrintMarkdown(resp)
	})

	if !strings.Contains(output, "Progress") {
		t.Fatalf("expected progress header in output, got: %s", output)
	}
	if !strings.Contains(output, "[####------] 3/7") {
		t.Fatalf("expected progress bar in output, got: %s", output)
	}
}
