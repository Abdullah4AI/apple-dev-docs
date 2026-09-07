package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebIAPTaxCategoryMutationResultsUseCamelCaseJSON(t *testing.T) {
	tests := []struct {
		name  string
		value any
		wants map[string]string
	}{
		{
			name: "set",
			value: &WebIAPTaxCategorySetResult{
				IAPID:        "iap-1",
				CategoryID:   "category-1",
				CategoryName: "Digital Goods",
				ConditionIDs: []string{"condition-1", "condition-2"},
				Changed:      true,
				Verified:     true,
			},
			wants: map[string]string{
				"iapId":        `"iap-1"`,
				"categoryId":   `"category-1"`,
				"categoryName": `"Digital Goods"`,
				"conditionIds": `["condition-1","condition-2"]`,
				"changed":      "true",
				"verified":     "true",
			},
		},
		{
			name: "reset",
			value: &WebIAPTaxCategoryResetResult{
				IAPID:    "iap-1",
				Changed:  false,
				Verified: true,
			},
			wants: map[string]string{
				"iapId":    `"iap-1"`,
				"changed":  "false",
				"verified": "true",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("json.Marshal() error: %v", err)
			}

			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}
			for field, want := range tc.wants {
				got, ok := fields[field]
				if !ok {
					t.Fatalf("JSON missing camelCase field %q: %s", field, encoded)
				}
				if string(got) != want {
					t.Fatalf("JSON field %q = %s, want %s", field, got, want)
				}
			}
			for _, field := range []string{"iap_id", "category_id", "category_name", "condition_ids"} {
				if _, ok := fields[field]; ok {
					t.Fatalf("JSON contains snake_case field %q: %s", field, encoded)
				}
			}
		})
	}
}

func TestWebIAPTaxCategoryMutationResultsUseRegisteredRenderers(t *testing.T) {
	results := []struct {
		name string
		data any
		want []string
	}{
		{
			name: "set",
			data: &WebIAPTaxCategorySetResult{
				IAPID:        "iap-1",
				CategoryID:   "category-1",
				CategoryName: "Digital Goods",
				ConditionIDs: []string{"condition-1", "condition-2"},
				Changed:      true,
				Verified:     true,
			},
			want: []string{
				"IAP ID", "Category ID", "Category Name", "Condition IDs", "Changed", "Verified",
				"iap-1", "category-1", "Digital Goods", "condition-1,condition-2", "true",
			},
		},
		{
			name: "reset",
			data: &WebIAPTaxCategoryResetResult{
				IAPID:    "iap-1",
				Changed:  false,
				Verified: true,
			},
			want: []string{"IAP ID", "Changed", "Verified", "iap-1", "false", "true"},
		},
	}

	ensureOutputRegistryPopulated()
	if !isRegistryTypeRegistered(typeForPtr[WebIAPTaxCategorySetResult]()) {
		t.Fatal("WebIAPTaxCategorySetResult is not registered with the output renderer")
	}
	if !isRegistryTypeRegistered(typeForPtr[WebIAPTaxCategoryResetResult]()) {
		t.Fatal("WebIAPTaxCategoryResetResult is not registered with the output renderer")
	}

	renderers := []struct {
		name string
		fn   func(any) error
	}{
		{name: "table", fn: PrintTable},
		{name: "markdown", fn: PrintMarkdown},
	}
	for _, result := range results {
		result := result
		t.Run(result.name, func(t *testing.T) {
			for _, renderer := range renderers {
				renderer := renderer
				t.Run(renderer.name, func(t *testing.T) {
					output := captureStdout(t, func() error { return renderer.fn(result.data) })
					if strings.TrimSpace(output) == "" {
						t.Fatal("expected rendered output")
					}
					for _, want := range result.want {
						if !strings.Contains(output, want) {
							t.Fatalf("output missing %q: %q", want, output)
						}
					}
				})
			}
		})
	}
}
