package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMatrixOutputJSONUsesGovernedCamelCase(t *testing.T) {
	result := &MatrixResult{
		PlanPath:   ".asc/matrix.json",
		BundleID:   "com.example.app",
		TotalCells: 1,
		Cells: []MatrixCellResult{{
			Content:      "default",
			DurationMS:   12,
			RawPaths:     []string{"raw/home.png"},
			FailureStage: "execution",
			FailureCode:  "plan_failed",
			Screenshots: []MatrixScreenshotResult{{
				RawPath: "raw/home.png",
			}},
			Steps: []MatrixStepResult{{DurationMS: 3}},
		}},
		Review: &MatrixReviewResult{ManifestPath: "review/manifest.json", HTMLPath: "review/index.html"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	encoded := string(data)
	for _, field := range []string{
		`"planPath"`, `"bundleId"`, `"totalCells"`, `"contentVariant"`, `"durationMs"`,
		`"rawPaths"`, `"failureStage"`, `"failureCode"`, `"rawPath"`, `"manifestPath"`, `"htmlPath"`,
	} {
		if !strings.Contains(encoded, field) {
			t.Fatalf("JSON output missing %s: %s", field, encoded)
		}
	}
	for _, field := range []string{
		`"plan_path"`, `"bundle_id"`, `"total_cells"`, `"content_variant"`, `"duration_ms"`,
		`"raw_paths"`, `"failure_stage"`, `"failure_code"`, `"raw_path"`, `"manifest_path"`, `"html_path"`,
	} {
		if strings.Contains(encoded, field) {
			t.Fatalf("JSON output contains legacy snake_case field %s: %s", field, encoded)
		}
	}
}

func TestMatrixOutputRegistryRendersSummaryCellsAndReview(t *testing.T) {
	result := &MatrixResult{
		PlanPath:   ".asc/matrix.json",
		Status:     "success",
		TotalCells: 1,
		Succeeded:  1,
		Cells: []MatrixCellResult{{
			ID:       "phone|en-US|light|default",
			Device:   "phone",
			Locale:   "en-US",
			Content:  "default",
			Status:   "success",
			Attempts: 1,
		}},
		Review: &MatrixReviewResult{
			ManifestPath: "review/manifest.json",
			HTMLPath:     "review/index.html",
		},
	}

	for _, renderer := range []struct {
		name string
		fn   func(any) error
	}{
		{name: "table", fn: PrintTable},
		{name: "markdown", fn: PrintMarkdown},
	} {
		t.Run(renderer.name, func(t *testing.T) {
			output := captureStdout(t, func() error { return renderer.fn(result) })
			for _, want := range []string{"Total Cells", "phone|en-US|light|default", "Manifest", "review/manifest.json"} {
				if !strings.Contains(output, want) {
					t.Fatalf("%s output missing %q: %s", renderer.name, want, output)
				}
			}
		})
	}
}

func TestMatrixReviewManifestJSONUsesGovernedCamelCase(t *testing.T) {
	manifest := &MatrixReviewManifest{
		GeneratedAt: "2026-08-30T00:00:00Z",
		PlanPath:    ".asc/matrix.json",
		TotalCells:  1,
		Cells: []MatrixCellResult{{
			Content:    "default",
			DurationMS: 5,
		}},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	encoded := string(data)
	for _, field := range []string{`"generatedAt"`, `"planPath"`, `"totalCells"`, `"contentVariant"`, `"durationMs"`} {
		if !strings.Contains(encoded, field) {
			t.Fatalf("manifest JSON missing %s: %s", field, encoded)
		}
	}
	for _, field := range []string{`"generated_at"`, `"plan_path"`, `"total_cells"`, `"content_variant"`, `"duration_ms"`} {
		if strings.Contains(encoded, field) {
			t.Fatalf("manifest JSON contains legacy snake_case field %s: %s", field, encoded)
		}
	}
}
