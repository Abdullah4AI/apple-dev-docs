package asc

import (
	"reflect"
	"testing"
)

func TestWebAppTaxCategorySetResultUsesOutputRegistry(t *testing.T) {
	result := &WebAppTaxCategorySetResult{
		AppID:        "app-1",
		CategoryID:   "category-1",
		CategoryName: "App Store Software",
		ConditionIDs: []string{"condition-1", "condition-2"},
		Changed:      true,
		Verified:     true,
	}

	var headers []string
	var rows [][]string
	if err := renderByRegistry(result, func(gotHeaders []string, gotRows [][]string) {
		headers = gotHeaders
		rows = gotRows
	}); err != nil {
		t.Fatalf("renderByRegistry() error: %v", err)
	}
	if want := []string{"Field", "Value"}; !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers = %#v, want %#v", headers, want)
	}
	if len(rows) != 6 || rows[0][1] != "app-1" || rows[3][1] != "condition-1,condition-2" || rows[5][1] != "true" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}

func TestWebAppTaxCategoryViewResultUsesOutputRegistry(t *testing.T) {
	result := &WebAppTaxCategoryViewResult{
		AppID:                 "app-1",
		Configured:            false,
		EffectiveCategoryID:   "category-default",
		EffectiveCategoryName: "App Store Software",
		EnabledConditionIDs:   []string{"condition-1"},
	}

	var headers []string
	var rows [][]string
	if err := renderByRegistry(result, func(gotHeaders []string, gotRows [][]string) {
		headers = gotHeaders
		rows = gotRows
	}); err != nil {
		t.Fatalf("renderByRegistry() error: %v", err)
	}
	if want := []string{"Field", "Value"}; !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers = %#v, want %#v", headers, want)
	}
	if len(rows) != 7 || rows[0][1] != "app-1" || rows[1][1] != "false" || rows[4][1] != "category-default" || rows[6][1] != "condition-1" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}
